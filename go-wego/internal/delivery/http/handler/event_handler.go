package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	eventUC "github.com/go-wego/wego/internal/usecase/event"
	"github.com/go-wego/wego/pkg/response"
	"github.com/go-wego/wego/pkg/validator"
	"github.com/google/uuid"
)

// EventHandler handles HTTP requests for event operations.
type EventHandler struct {
	uc eventUC.UseCase
}

// NewEventHandler constructs an EventHandler.
func NewEventHandler(uc eventUC.UseCase) *EventHandler {
	return &EventHandler{uc: uc}
}

// ——— Create ——————————————————————————————————————————————————————————————————

// CreateEvent godoc
// @Summary      Create a new event
// @Tags         events
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body event.CreateInput true "Event payload"
// @Success      201  {object} entity.Event
// @Router       /api/v1/events [post]
func (h *EventHandler) CreateEvent(c *gin.Context) {
	hostID, err := extractUserID(c)
	if err != nil {
		response.Unauthorized(c, "invalid user context")
		return
	}

	var in eventUC.CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "invalid JSON body")
		return
	}
	if err := validator.Validate(in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	e, err := h.uc.Create(c.Request.Context(), hostID, in)
	if err != nil {
		if errors.Is(err, eventUC.ErrInvalidTimeRange) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c, "failed to create event")
		return
	}

	response.Created(c, e)
}

// ——— GetEvent ————————————————————————————————————————————————————————————————

// GetEvent godoc
// @Summary      Get event by ID
// @Tags         events
// @Produce      json
// @Param        id   path    string  true  "Event UUID"
// @Success      200  {object} entity.Event
// @Router       /api/v1/events/{id} [get]
func (h *EventHandler) GetEvent(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid event id")
		return
	}

	e, err := h.uc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, eventUC.ErrNotFound) {
			response.NotFound(c, "event not found")
			return
		}
		response.InternalError(c, "failed to get event")
		return
	}

	response.OK(c, e)
}

// ——— ListEvents ——————————————————————————————————————————————————————————————

// ListEvents godoc
// @Summary      List events with pagination and filters
// @Tags         events
// @Produce      json
// @Param        page      query  int     false  "Page number (default 1)"
// @Param        limit     query  int     false  "Page size (default 20, max 100)"
// @Param        status    query  string  false  "Filter by status"
// @Param        location  query  string  false  "Filter by location"
// @Param        search    query  string  false  "Search in title/description"
// @Success      200  {object} event.ListResult
// @Router       /api/v1/events [get]
func (h *EventHandler) ListEvents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit > 100 {
		limit = 100
	}

	in := eventUC.ListInput{
		Page:     page,
		Limit:    limit,
		Status:   c.Query("status"),
		Location: c.Query("location"),
		Search:   c.Query("search"),
	}

	result, err := h.uc.List(c.Request.Context(), in)
	if err != nil {
		response.InternalError(c, "failed to list events")
		return
	}

	response.OK(c, result)
}

// ——— JoinEvent ———————————————————————————————————————————————————————————————

// JoinEvent godoc
// @Summary      Join an event
// @Tags         events
// @Security     BearerAuth
// @Param        id   path    string  true  "Event UUID"
// @Success      200
// @Router       /api/v1/events/{id}/join [post]
func (h *EventHandler) JoinEvent(c *gin.Context) {
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid event id")
		return
	}

	userID, err := extractUserID(c)
	if err != nil {
		response.Unauthorized(c, "invalid user context")
		return
	}

	if err := h.uc.Join(c.Request.Context(), eventID, userID); err != nil {
		switch {
		case errors.Is(err, eventUC.ErrNotFound):
			response.NotFound(c, "event not found")
		case errors.Is(err, eventUC.ErrEventFull):
			response.BadRequest(c, err.Error())
		case errors.Is(err, eventUC.ErrAlreadyJoined):
			response.Conflict(c, err.Error())
		case errors.Is(err, eventUC.ErrEventNotJoinable):
			response.BadRequest(c, err.Error())
		default:
			response.InternalError(c, "failed to join event")
		}
		return
	}

	response.OK(c, gin.H{"message": "joined event successfully"})
}

// ——— LeaveEvent ——————————————————————————————————————————————————————————————

// LeaveEvent godoc
// @Summary      Leave an event
// @Tags         events
// @Security     BearerAuth
// @Param        id   path    string  true  "Event UUID"
// @Success      200
// @Router       /api/v1/events/{id}/leave [post]
func (h *EventHandler) LeaveEvent(c *gin.Context) {
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid event id")
		return
	}

	userID, err := extractUserID(c)
	if err != nil {
		response.Unauthorized(c, "invalid user context")
		return
	}

	if err := h.uc.Leave(c.Request.Context(), eventID, userID); err != nil {
		switch {
		case errors.Is(err, eventUC.ErrNotParticipant):
			response.BadRequest(c, err.Error())
		default:
			response.InternalError(c, "failed to leave event")
		}
		return
	}

	response.OK(c, gin.H{"message": "left event successfully"})
}

// ——— DeleteEvent ——————————————————————————————————————————————————————————————————

// DeleteEvent godoc
// @Summary      Soft-delete an event (host only)
// @Tags         events
// @Security     BearerAuth
// @Param        id   path    string  true  "Event UUID"
// @Success      204
// @Router       /api/v1/events/{id} [delete]
func (h *EventHandler) DeleteEvent(c *gin.Context) {
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid event id")
		return
	}

	userID, err := extractUserID(c)
	if err != nil {
		response.Unauthorized(c, "invalid user context")
		return
	}

	if err := h.uc.Delete(c.Request.Context(), eventID, userID); err != nil {
		switch {
		case errors.Is(err, eventUC.ErrNotFound):
			response.NotFound(c, "event not found")
		case errors.Is(err, eventUC.ErrForbidden):
			response.Forbidden(c, err.Error())
		default:
			response.InternalError(c, "failed to delete event")
		}
		return
	}

	response.NoContent(c)
}
