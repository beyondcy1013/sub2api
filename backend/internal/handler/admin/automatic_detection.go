package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/modules/autodetection"
	"github.com/gin-gonic/gin"
)

func (h *ScheduledTestHandler) GetAutomaticDetection(c *gin.Context) {
	autodetection.Handler{Engine: h.scheduledTestSvc.AutomaticDetection()}.Get(c)
}
func (h *ScheduledTestHandler) UpdateAutomaticDetection(c *gin.Context) {
	autodetection.Handler{Engine: h.scheduledTestSvc.AutomaticDetection()}.Update(c)
}
