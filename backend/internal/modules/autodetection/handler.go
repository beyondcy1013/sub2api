package autodetection

import (
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

var ErrInvalidConfig = errors.New("invalid automatic detection policy")

// Handler is mounted only inside the existing authenticated administrator group.
type Handler struct{ Engine *Engine }

func (h Handler) Get(c *gin.Context) {
	if h.Engine == nil {
		response.Error(c, 503, "automatic detection unavailable")
		return
	}
	config, err := h.Engine.Config(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	runtime, err := h.Engine.Runtime(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"config": config, "runtime": runtime})
}
func (h Handler) Update(c *gin.Context) {
	if h.Engine == nil {
		response.Error(c, 503, "automatic detection unavailable")
		return
	}
	var config Config
	if err := c.ShouldBindJSON(&config); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.Engine.Save(c.Request.Context(), config); err != nil {
		if errors.Is(err, ErrInvalidConfig) {
			response.BadRequest(c, err.Error())
		} else {
			response.InternalError(c, err.Error())
		}
		return
	}
	h.Get(c)
}
