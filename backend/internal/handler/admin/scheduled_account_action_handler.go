package admin

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type scheduledAccountActionManager interface {
	Schedule(ctx context.Context, accountID int64, action service.ScheduledAccountActionType, delay time.Duration) (*service.ScheduledAccountAction, error)
	GetScheduledAction(ctx context.Context, accountID int64) (*service.ScheduledAccountAction, error)
	CancelScheduledAction(ctx context.Context, accountID int64) error
}

type ScheduledAccountActionHandler struct {
	service scheduledAccountActionManager
}

func NewScheduledAccountActionHandler(actionService *service.ScheduledAccountActionService) *ScheduledAccountActionHandler {
	return &ScheduledAccountActionHandler{service: actionService}
}

type upsertScheduledAccountActionRequest struct {
	Action  service.ScheduledAccountActionType `json:"action"`
	Hours   int                                `json:"hours"`
	Minutes int                                `json:"minutes"`
}

func scheduledAccountID(c *gin.Context) (int64, bool) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || accountID <= 0 {
		response.BadRequest(c, "Invalid account ID")
		return 0, false
	}
	return accountID, true
}

func (h *ScheduledAccountActionHandler) Get(c *gin.Context) {
	accountID, ok := scheduledAccountID(c)
	if !ok {
		return
	}
	action, err := h.service.GetScheduledAction(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, action)
}

func (h *ScheduledAccountActionHandler) Upsert(c *gin.Context) {
	accountID, ok := scheduledAccountID(c)
	if !ok {
		return
	}
	var request upsertScheduledAccountActionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if request.Hours < 0 || request.Hours > 8760 || request.Minutes < 0 || request.Minutes > 59 {
		response.BadRequest(c, service.ErrInvalidScheduledActionDelay.Error())
		return
	}
	if !request.Action.Valid() {
		response.BadRequest(c, service.ErrInvalidScheduledAccountAction.Error())
		return
	}
	delay := time.Duration(request.Hours)*time.Hour + time.Duration(request.Minutes)*time.Minute
	if delay < time.Minute || delay > 365*24*time.Hour {
		response.BadRequest(c, service.ErrInvalidScheduledActionDelay.Error())
		return
	}
	action, err := h.service.Schedule(c.Request.Context(), accountID, request.Action, delay)
	if err != nil {
		if errors.Is(err, service.ErrInvalidScheduledAccountAction) || errors.Is(err, service.ErrInvalidScheduledActionDelay) {
			response.BadRequest(c, err.Error())
		} else {
			response.ErrorFrom(c, err)
		}
		return
	}
	response.Success(c, action)
}

func (h *ScheduledAccountActionHandler) Delete(c *gin.Context) {
	accountID, ok := scheduledAccountID(c)
	if !ok {
		return
	}
	if err := h.service.CancelScheduledAction(c.Request.Context(), accountID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Scheduled account action canceled"})
}
