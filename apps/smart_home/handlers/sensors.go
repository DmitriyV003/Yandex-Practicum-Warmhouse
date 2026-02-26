package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"smarthome/db"
	"smarthome/models"
	"smarthome/services"

	"github.com/gin-gonic/gin"
)

// SensorHandler handles sensor-related requests
type SensorHandler struct {
	DB                 *db.DB
	TemperatureService *services.TemperatureService
	DeviceClient       *services.DeviceClient
	TelemetryClient    *services.TelemetryClient
}

// NewSensorHandler creates a new SensorHandler
func NewSensorHandler(db *db.DB, temperatureService *services.TemperatureService, deviceClient *services.DeviceClient, telemetryClient *services.TelemetryClient) *SensorHandler {
	return &SensorHandler{
		DB:                 db,
		TemperatureService: temperatureService,
		DeviceClient:       deviceClient,
		TelemetryClient:    telemetryClient,
	}
}

// RegisterRoutes registers the sensor routes
func (h *SensorHandler) RegisterRoutes(router *gin.RouterGroup) {
	sensors := router.Group("/sensors")
	{
		sensors.GET("", h.GetSensors)
		sensors.GET("/:id", h.GetSensorByID)
		sensors.POST("", h.CreateSensor)
		sensors.PUT("/:id", h.UpdateSensor)
		sensors.DELETE("/:id", h.DeleteSensor)
		sensors.PATCH("/:id/value", h.UpdateSensorValue)
		sensors.GET("/temperature/:location", h.GetTemperatureByLocation)
	}
}

// GetSensors godoc
// @Summary      Получить все датчики
// @Description  Возвращает список всех датчиков с актуальными данными температуры
// @Tags         sensors
// @Produce      json
// @Success      200 {array} models.Sensor
// @Failure      500 {object} map[string]string
// @Router       /sensors [get]
func (h *SensorHandler) GetSensors(c *gin.Context) {
	sensors, err := h.DB.GetSensors(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update temperature sensors with real-time data
	for i, sensor := range sensors {
		if sensor.Type == models.Temperature {
			// Try telemetry-service first (microservice), fall back to direct temperature API
			if h.TelemetryClient != nil {
				resp, err := h.TelemetryClient.CollectFromSensor(context.Background(), int32(sensor.ID), sensor.Location)
				if err == nil {
					sensors[i].Value = resp.Value
					sensors[i].Status = "active"
					sensors[i].LastUpdated = resp.CreatedAt.AsTime()
					log.Printf("Updated sensor %d via telemetry-service", sensor.ID)
					continue
				}
				log.Printf("telemetry-service failed for sensor %d, falling back: %v", sensor.ID, err)
			}
			// Fallback to direct temperature API
			tempData, err := h.TemperatureService.GetTemperatureByID(fmt.Sprintf("%d", sensor.ID))
			if err == nil {
				sensors[i].Value = tempData.Value
				sensors[i].Status = tempData.Status
				sensors[i].LastUpdated = tempData.Timestamp
				log.Printf("Updated sensor %d from temperature API (fallback)", sensor.ID)
			} else {
				log.Printf("Failed to fetch temperature for sensor %d: %v", sensor.ID, err)
			}
		}
	}

	c.JSON(http.StatusOK, sensors)
}

// GetSensorByID godoc
// @Summary      Получить датчик по ID
// @Description  Возвращает датчик с актуальными данными температуры
// @Tags         sensors
// @Produce      json
// @Param        id path int true "ID датчика"
// @Success      200 {object} models.Sensor
// @Failure      400 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /sensors/{id} [get]
func (h *SensorHandler) GetSensorByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sensor ID"})
		return
	}

	sensor, err := h.DB.GetSensorByID(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sensor not found"})
		return
	}

	// If this is a temperature sensor, fetch real-time data from the temperature API
	if sensor.Type == models.Temperature {
		tempData, err := h.TemperatureService.GetTemperatureByID(fmt.Sprintf("%d", sensor.ID))
		if err == nil {
			// Update sensor with real-time data
			sensor.Value = tempData.Value
			sensor.Status = tempData.Status
			sensor.LastUpdated = tempData.Timestamp
			log.Printf("Updated temperature data for sensor %d from external API", sensor.ID)
		} else {
			log.Printf("Failed to fetch temperature data for sensor %d: %v", sensor.ID, err)
		}
	}

	c.JSON(http.StatusOK, sensor)
}

// GetTemperatureByLocation godoc
// @Summary      Получить температуру по локации
// @Description  Возвращает текущую температуру для указанной локации
// @Tags         temperature
// @Produce      json
// @Param        location path string true "Название локации (Living Room, Bedroom, Kitchen)"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /sensors/temperature/{location} [get]
func (h *SensorHandler) GetTemperatureByLocation(c *gin.Context) {
	location := c.Param("location")
	if location == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Location is required"})
		return
	}

	// Fetch temperature data from the external API
	tempData, err := h.TemperatureService.GetTemperature(location)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to fetch temperature data: %v", err),
		})
		return
	}

	// Return the temperature data
	c.JSON(http.StatusOK, gin.H{
		"location":    tempData.Location,
		"value":       tempData.Value,
		"unit":        tempData.Unit,
		"status":      tempData.Status,
		"timestamp":   tempData.Timestamp,
		"description": tempData.Description,
	})
}

// CreateSensor godoc
// @Summary      Создать датчик
// @Description  Создает новый датчик и зеркалирует в device-service
// @Tags         sensors
// @Accept       json
// @Produce      json
// @Param        sensor body models.SensorCreate true "Данные датчика"
// @Success      201 {object} models.Sensor
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /sensors [post]
func (h *SensorHandler) CreateSensor(c *gin.Context) {
	var sensorCreate models.SensorCreate
	if err := c.ShouldBindJSON(&sensorCreate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sensor, err := h.DB.CreateSensor(context.Background(), sensorCreate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Mirror to device-service via gRPC (non-fatal)
	if h.DeviceClient != nil {
		if _, err := h.DeviceClient.CreateDevice(context.Background(),
			sensorCreate.Name, string(sensorCreate.Type),
			sensorCreate.Unit, sensorCreate.Location); err != nil {
			log.Printf("device-service CreateDevice failed (continuing): %v", err)
		} else {
			log.Printf("Mirrored sensor %d to device-service", sensor.ID)
		}
	}

	c.JSON(http.StatusCreated, sensor)
}

// UpdateSensor godoc
// @Summary      Обновить датчик
// @Description  Обновляет данные существующего датчика
// @Tags         sensors
// @Accept       json
// @Produce      json
// @Param        id path int true "ID датчика"
// @Param        sensor body models.SensorUpdate true "Данные для обновления"
// @Success      200 {object} models.Sensor
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /sensors/{id} [put]
func (h *SensorHandler) UpdateSensor(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sensor ID"})
		return
	}

	var sensorUpdate models.SensorUpdate
	if err := c.ShouldBindJSON(&sensorUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sensor, err := h.DB.UpdateSensor(context.Background(), id, sensorUpdate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sensor)
}

// DeleteSensor godoc
// @Summary      Удалить датчик
// @Description  Удаляет датчик и зеркалирует удаление в device-service
// @Tags         sensors
// @Produce      json
// @Param        id path int true "ID датчика"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /sensors/{id} [delete]
func (h *SensorHandler) DeleteSensor(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sensor ID"})
		return
	}

	err = h.DB.DeleteSensor(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Mirror to device-service via gRPC (non-fatal)
	if h.DeviceClient != nil {
		if err := h.DeviceClient.DeleteDevice(context.Background(), int32(id)); err != nil {
			log.Printf("device-service DeleteDevice failed (continuing): %v", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sensor deleted successfully"})
}

// UpdateSensorValue godoc
// @Summary      Обновить значение датчика
// @Description  Обновляет текущее значение и статус датчика
// @Tags         sensors
// @Accept       json
// @Produce      json
// @Param        id path int true "ID датчика"
// @Param        value body object true "Значение и статус" example({"value": 23.5, "status": "active"})
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /sensors/{id}/value [patch]
func (h *SensorHandler) UpdateSensorValue(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sensor ID"})
		return
	}

	var request struct {
		Value  float64 `json:"value" binding:"required"`
		Status string  `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.DB.UpdateSensorValue(context.Background(), id, request.Value, request.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sensor value updated successfully"})
}
