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

type DeviceHandler struct {
	DB                 *db.DB
	TemperatureService *services.TemperatureService
	DeviceClient       *services.DeviceClient
	TelemetryClient    *services.TelemetryClient
}

func NewSensorHandler(db *db.DB, temperatureService *services.TemperatureService, deviceClient *services.DeviceClient, telemetryClient *services.TelemetryClient) *DeviceHandler {
	return &DeviceHandler{
		DB:                 db,
		TemperatureService: temperatureService,
		DeviceClient:       deviceClient,
		TelemetryClient:    telemetryClient,
	}
}

func (h *DeviceHandler) RegisterRoutes(router *gin.RouterGroup) {
	devices := router.Group("/devices")
	{
		devices.GET("", h.GetDevices)
		devices.GET("/:id", h.GetDeviceByID)
		devices.POST("", h.CreateDevice)
		devices.PUT("/:id", h.UpdateDevice)
		devices.DELETE("/:id", h.DeleteDevice)
		devices.GET("/temperature/:room", h.GetTemperatureByRoom)
	}

	rooms := router.Group("/rooms")
	{
		rooms.GET("", h.GetRooms)
		rooms.POST("", h.CreateRoom)
	}

	houses := router.Group("/houses")
	{
		houses.GET("", h.GetHouses)
		houses.POST("", h.CreateHouse)
	}

	users := router.Group("/users")
	{
		users.POST("", h.CreateUser)
	}
}

// GetDevices godoc
// @Summary      Получить все устройства
// @Description  Возвращает список всех устройств с актуальными данными температуры
// @Tags         devices
// @Produce      json
// @Success      200 {array} models.Device
// @Failure      500 {object} map[string]string
// @Router       /devices [get]
func (h *DeviceHandler) GetDevices(c *gin.Context) {
	devices, err := h.DB.GetDevices(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for i, device := range devices {
		if device.Type == models.Temperature {
			if h.TelemetryClient != nil {
				_, err := h.TelemetryClient.CollectFromSensor(context.Background(), int32(device.ID), device.RoomName)
				if err == nil {
					devices[i].Status = "active"
					log.Printf("Updated device %d via telemetry-service", device.ID)
					continue
				}
				log.Printf("telemetry-service failed for device %d, falling back: %v", device.ID, err)
			}
			tempData, err := h.TemperatureService.GetTemperatureByID(fmt.Sprintf("%d", device.ID))
			if err == nil {
				devices[i].Status = tempData.Status
				log.Printf("Updated device %d from temperature API (fallback)", device.ID)
			} else {
				log.Printf("Failed to fetch temperature for device %d: %v", device.ID, err)
			}
		}
	}

	c.JSON(http.StatusOK, devices)
}

// GetDeviceByID godoc
// @Summary      Получить устройство по ID
// @Description  Возвращает устройство с актуальными данными
// @Tags         devices
// @Produce      json
// @Param        id path int true "ID устройства"
// @Success      200 {object} models.Device
// @Failure      400 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /devices/{id} [get]
func (h *DeviceHandler) GetDeviceByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}

	device, err := h.DB.GetDeviceByID(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	if device.Type == models.Temperature {
		tempData, err := h.TemperatureService.GetTemperatureByID(fmt.Sprintf("%d", device.ID))
		if err == nil {
			device.Status = tempData.Status
			log.Printf("Updated temperature data for device %d from external API", device.ID)
		} else {
			log.Printf("Failed to fetch temperature data for device %d: %v", device.ID, err)
		}
	}

	c.JSON(http.StatusOK, device)
}

// GetTemperatureByRoom godoc
// @Summary      Получить температуру по комнате
// @Description  Возвращает текущую температуру для указанной комнаты
// @Tags         temperature
// @Produce      json
// @Param        room path string true "Название комнаты"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /devices/temperature/{room} [get]
func (h *DeviceHandler) GetTemperatureByRoom(c *gin.Context) {
	room := c.Param("room")
	if room == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Room is required"})
		return
	}

	tempData, err := h.TemperatureService.GetTemperature(room)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to fetch temperature data: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"room":        tempData.Location,
		"value":       tempData.Value,
		"unit":        tempData.Unit,
		"status":      tempData.Status,
		"timestamp":   tempData.Timestamp,
		"description": tempData.Description,
	})
}

// CreateDevice godoc
// @Summary      Создать устройство
// @Description  Создает новое устройство и зеркалирует в device-service
// @Tags         devices
// @Accept       json
// @Produce      json
// @Param        device body models.DeviceCreate true "Данные устройства"
// @Success      201 {object} models.Device
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /devices [post]
func (h *DeviceHandler) CreateDevice(c *gin.Context) {
	var deviceCreate models.DeviceCreate
	if err := c.ShouldBindJSON(&deviceCreate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	device, err := h.DB.CreateDevice(context.Background(), deviceCreate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.DeviceClient != nil {
		if _, err := h.DeviceClient.CreateDevice(context.Background(),
			deviceCreate.Name, string(deviceCreate.Type),
			deviceCreate.Unit, int32(deviceCreate.RoomID)); err != nil {
			log.Printf("device-service CreateDevice failed (continuing): %v", err)
		} else {
			log.Printf("Mirrored device %d to device-service", device.ID)
		}
	}

	c.JSON(http.StatusCreated, device)
}

// UpdateDevice godoc
// @Summary      Обновить устройство
// @Description  Обновляет данные существующего устройства
// @Tags         devices
// @Accept       json
// @Produce      json
// @Param        id path int true "ID устройства"
// @Param        device body models.DeviceUpdate true "Данные для обновления"
// @Success      200 {object} models.Device
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /devices/{id} [put]
func (h *DeviceHandler) UpdateDevice(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}

	var deviceUpdate models.DeviceUpdate
	if err := c.ShouldBindJSON(&deviceUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	device, err := h.DB.UpdateDevice(context.Background(), id, deviceUpdate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, device)
}

// DeleteDevice godoc
// @Summary      Удалить устройство
// @Description  Удаляет устройство и зеркалирует удаление в device-service
// @Tags         devices
// @Produce      json
// @Param        id path int true "ID устройства"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /devices/{id} [delete]
func (h *DeviceHandler) DeleteDevice(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}

	err = h.DB.DeleteDevice(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.DeviceClient != nil {
		if err := h.DeviceClient.DeleteDevice(context.Background(), int32(id)); err != nil {
			log.Printf("device-service DeleteDevice failed (continuing): %v", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Device deleted successfully"})
}

// GetRooms godoc
// @Summary      Получить комнаты
// @Tags         rooms
// @Produce      json
// @Param        house_id query int true "ID дома"
// @Success      200 {array} models.Room
// @Router       /rooms [get]
func (h *DeviceHandler) GetRooms(c *gin.Context) {
	houseID, err := strconv.Atoi(c.Query("house_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "house_id is required"})
		return
	}

	rooms, err := h.DB.GetRooms(context.Background(), houseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rooms)
}

// CreateRoom godoc
// @Summary      Создать комнату
// @Tags         rooms
// @Accept       json
// @Produce      json
// @Param        room body models.RoomCreate true "Данные комнаты"
// @Success      201 {object} models.Room
// @Router       /rooms [post]
func (h *DeviceHandler) CreateRoom(c *gin.Context) {
	var roomCreate models.RoomCreate
	if err := c.ShouldBindJSON(&roomCreate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	room, err := h.DB.CreateRoom(context.Background(), roomCreate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, room)
}

// GetHouses godoc
// @Summary      Получить дома
// @Tags         houses
// @Produce      json
// @Param        owner_id query int true "ID владельца"
// @Success      200 {array} models.House
// @Router       /houses [get]
func (h *DeviceHandler) GetHouses(c *gin.Context) {
	ownerID, err := strconv.Atoi(c.Query("owner_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "owner_id is required"})
		return
	}

	houses, err := h.DB.GetHouses(context.Background(), ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, houses)
}

// CreateHouse godoc
// @Summary      Создать дом
// @Tags         houses
// @Accept       json
// @Produce      json
// @Param        house body models.HouseCreate true "Данные дома"
// @Success      201 {object} models.House
// @Router       /houses [post]
func (h *DeviceHandler) CreateHouse(c *gin.Context) {
	var houseCreate models.HouseCreate
	if err := c.ShouldBindJSON(&houseCreate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	house, err := h.DB.CreateHouse(context.Background(), houseCreate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, house)
}

// CreateUser godoc
// @Summary      Создать пользователя
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        user body models.UserCreate true "Данные пользователя"
// @Success      201 {object} models.User
// @Router       /users [post]
func (h *DeviceHandler) CreateUser(c *gin.Context) {
	var userCreate models.UserCreate
	if err := c.ShouldBindJSON(&userCreate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.DB.CreateUser(context.Background(), userCreate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, user)
}
