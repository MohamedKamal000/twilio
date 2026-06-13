package handlers

import (
	"oba-twilio/client"
	"oba-twilio/handlers/common"
	"oba-twilio/handlers/voice"
	"oba-twilio/localization"
	"oba-twilio/middleware"

	"github.com/gin-gonic/gin"
)

type VoiceHandler struct {
	*voice.Handler
}

func NewVoiceHandler(obaClient client.OneBusAwayClientInterface, locManager *localization.LocalizationManager) *VoiceHandler {
	return &VoiceHandler{
		Handler: voice.NewHandler(obaClient, locManager),
	}
}

func (h *VoiceHandler) Close() {
	if h.Handler != nil {
		h.Handler.Close()
	}
}

func (h *VoiceHandler) SetAnalytics(analyticsManager middleware.AnalyticsManager, hashSalt string) {
	if h.Handler != nil {
		h.Handler.SetAnalytics(analyticsManager, hashSalt)
	}
}

func (h *VoiceHandler) SetArrivalFilterConfig(cfg common.ArrivalFilterConfig) {
	if h.Handler != nil {
		h.Handler.SetArrivalFilterConfig(cfg)
	}
}

func (h *VoiceHandler) HandleVoiceMenuAction(c *gin.Context) {
	if h.Handler != nil {
		h.Handler.HandleVoiceMenuAction(c)
	}
}

func (h *VoiceHandler) HandleVoiceStart(c *gin.Context) {
	if h.Handler != nil {
		h.Handler.HandleVoiceStart(c)
	}
}

func (h *VoiceHandler) HandleFindStop(c *gin.Context) {
	if h.Handler != nil {
		h.Handler.HandleFindStop(c)
	}
}
