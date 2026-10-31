package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"smarthome/db"
	"smarthome/models"
	"smarthome/services"

	"github.com/gin-gonic/gin"
)

// SensorHandler handles sensor-related requests
type SensorHandler struct {
	DB                 *db.DB
	TemperatureService *services.TemperatureService
	DeviceService      *services.DeviceService
	TelemetryService   *services.TelemetryService
	EventBus           *services.EventBus
}

// NewSensorHandler creates a new SensorHandler
func NewSensorHandler(
	db *db.DB,
	temperatureService *services.TemperatureService,
	deviceService *services.DeviceService,
	telemetryService *services.TelemetryService,
	eventBus *services.EventBus,
) *SensorHandler {
	return &SensorHandler{
		DB:                 db,
		TemperatureService: temperatureService,
		DeviceService:      deviceService,
		TelemetryService:   telemetryService,
		EventBus:           eventBus,
	}
}

// RegisterRoutes registers the sensor routes
func (h *SensorHandler) RegisterRoutes(router *gin.RouterGroup) {
	sensors := router.Group("/sensors")
	{
		sensors.GET("", h.GetSensors)
		sensors.GET("/:id", h.GetSensorByID)
		sensors.GET("/:id/telemetry", h.GetSensorTelemetry)
		sensors.POST("", h.CreateSensor)
		sensors.PUT("/:id", h.UpdateSensor)
		sensors.DELETE("/:id", h.DeleteSensor)
		sensors.PATCH("/:id/value", h.UpdateSensorValue)
		sensors.GET("/temperature/:location", h.GetTemperatureByLocation)
	}
}

// GetSensors handles GET /api/v1/sensors
func (h *SensorHandler) GetSensors(c *gin.Context) {
	sensors, err := h.DB.GetSensors(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update temperature sensors with real-time data from the external API
	for i, sensor := range sensors {
		if sensor.Type == models.Temperature {
			tempData, err := h.TemperatureService.GetTemperatureByID(fmt.Sprintf("%d", sensor.ID))
			if err == nil {
				// Update sensor with real-time data
				sensors[i].Value = tempData.Value
				sensors[i].Status = tempData.Status
				sensors[i].LastUpdated = tempData.Timestamp
				log.Printf("Updated temperature data for sensor %d from external API", sensor.ID)
			} else {
				log.Printf("Failed to fetch temperature data for sensor %d: %v", sensor.ID, err)
			}
		}

		h.syncDevice(sensors[i])
		h.recordTelemetry(sensors[i])
	}

	c.JSON(http.StatusOK, sensors)
}

// GetSensorByID handles GET /api/v1/sensors/:id
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

	h.syncDevice(sensor)
	h.recordTelemetry(sensor)

	c.JSON(http.StatusOK, sensor)
}

// GetTemperatureByLocation handles GET /api/v1/sensors/temperature/:location
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

// CreateSensor handles POST /api/v1/sensors
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

	h.syncDevice(sensor)

	c.JSON(http.StatusCreated, sensor)
}

// UpdateSensor handles PUT /api/v1/sensors/:id
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

	h.syncDevice(sensor)

	c.JSON(http.StatusOK, sensor)
}

// DeleteSensor handles DELETE /api/v1/sensors/:id
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

	h.deleteDevice(fmt.Sprintf("%d", id))

	c.JSON(http.StatusOK, gin.H{"message": "Sensor deleted successfully"})
}

// UpdateSensorValue handles PATCH /api/v1/sensors/:id/value
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

	sensor, err := h.DB.GetSensorByID(context.Background(), id)
	if err == nil {
		h.syncDevice(sensor)
		h.recordTelemetry(sensor)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sensor value updated successfully"})
}

// GetSensorTelemetry handles GET /api/v1/sensors/:id/telemetry
func (h *SensorHandler) GetSensorTelemetry(c *gin.Context) {
	id := c.Param("id")
	if _, err := strconv.Atoi(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sensor ID"})
		return
	}

	if h.TelemetryService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Telemetry service is not configured"})
		return
	}

	limit := 50
	if rawLimit := c.Query("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a positive integer"})
			return
		}
		limit = parsedLimit
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	readings, err := h.TelemetryService.List(ctx, id, limit)
	if err != nil {
		log.Printf("Failed to fetch telemetry for sensor %s: %v", id, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Telemetry service is unavailable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": readings,
		"total": len(readings),
	})
}

func (h *SensorHandler) syncDevice(sensor models.Sensor) {
	if h.DeviceService != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := h.DeviceService.Upsert(ctx, services.DevicePayload{
			ExternalID: fmt.Sprintf("%d", sensor.ID),
			Name:       sensor.Name,
			Type:       string(sensor.Type),
			Location:   sensor.Location,
			Unit:       sensor.Unit,
			Status:     sensor.Status,
			State: map[string]any{
				"value": sensor.Value,
				"unit":  sensor.Unit,
			},
		})
		if err != nil {
			log.Printf("Failed to sync sensor %d with device-service: %v", sensor.ID, err)
		}
	}

	if h.EventBus != nil {
		eventCtx, eventCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer eventCancel()
		if err := h.EventBus.PublishDeviceRegistered(eventCtx, services.DeviceRegisteredEvent{
			ExternalID: fmt.Sprintf("%d", sensor.ID),
			Name:       sensor.Name,
			Type:       string(sensor.Type),
			Location:   sensor.Location,
			Unit:       sensor.Unit,
			Status:     sensor.Status,
			State: map[string]any{
				"value": sensor.Value,
				"unit":  sensor.Unit,
			},
		}); err != nil {
			log.Printf("Failed to publish device registration for sensor %d: %v", sensor.ID, err)
		}
	}
}

func (h *SensorHandler) recordTelemetry(sensor models.Sensor) {
	if sensor.Type != models.Temperature {
		return
	}

	if h.EventBus != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := h.EventBus.PublishTelemetry(ctx, services.TelemetryRecordedEvent{
			DeviceID:   fmt.Sprintf("%d", sensor.ID),
			Metric:     "temperature",
			Value:      sensor.Value,
			Unit:       sensor.Unit,
			RecordedAt: sensor.LastUpdated,
		}); err != nil {
			log.Printf("Failed to publish telemetry for sensor %d: %v", sensor.ID, err)
		}
		return
	}

	if h.TelemetryService != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := h.TelemetryService.Record(ctx, services.TelemetryRecord{
			DeviceID:   fmt.Sprintf("%d", sensor.ID),
			Metric:     "temperature",
			Value:      sensor.Value,
			Unit:       sensor.Unit,
			RecordedAt: sensor.LastUpdated,
		}); err != nil {
			log.Printf("Failed to record telemetry for sensor %d: %v", sensor.ID, err)
		}
	}
}

func (h *SensorHandler) deleteDevice(externalID string) {
	if h.DeviceService == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := h.DeviceService.Delete(ctx, externalID); err != nil {
		log.Printf("Failed to delete device %s from device-service: %v", externalID, err)
	}
}
