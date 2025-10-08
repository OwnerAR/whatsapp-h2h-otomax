package handler

import (
	"encoding/json"
	"net/http"

	"whatsapp-h2h-otomax/internal/service"
	"whatsapp-h2h-otomax/pkg/logger"
)

// GroupsHandler handles group-related requests
type GroupsHandler struct {
	whatsappService *service.WhatsAppService
	logger          *logger.Logger
}

// NewGroupsHandler creates a new groups handler
func NewGroupsHandler(waService *service.WhatsAppService, log *logger.Logger) *GroupsHandler {
	return &GroupsHandler{
		whatsappService: waService,
		logger:          log,
	}
}

// GroupInfo represents group information for API response
type GroupInfo struct {
	JID          string `json:"jid"`
	Name         string `json:"name"`
	Topic        string `json:"topic,omitempty"`
	Participants int    `json:"participants"`
	IsAnnounce   bool   `json:"is_announce"`
}

// GetGroupsResponse represents the API response
type GetGroupsResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    []GroupInfo `json:"data"`
}

// ListGroups handles GET /api/v1/groups
func (h *GroupsHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	groups, err := h.whatsappService.GetJoinedGroups(ctx)
	if err != nil {
		h.logger.Error("Failed to get joined groups", "error", err)
		h.sendErrorResponse(w, "Failed to retrieve groups", http.StatusInternalServerError)
		return
	}

	// Convert to API response format
	groupsList := make([]GroupInfo, 0, len(groups))
	for _, group := range groups {
		groupInfo := GroupInfo{
			JID:          group.JID.String(),
			Name:         group.Name,
			Participants: len(group.Participants),
		}

		// Get additional info
		info, err := h.whatsappService.GetClient().GetGroupInfo(group.JID)
		if err == nil {
			groupInfo.Topic = info.Topic
			groupInfo.IsAnnounce = info.IsAnnounce
		}

		groupsList = append(groupsList, groupInfo)
	}

	h.logger.Info("Groups list retrieved", "total", len(groupsList))
	h.sendSuccessResponse(w, groupsList)
}

// sendSuccessResponse sends success response
func (h *GroupsHandler) sendSuccessResponse(w http.ResponseWriter, data []GroupInfo) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := GetGroupsResponse{
		Status:  "success",
		Message: "Groups retrieved successfully",
		Data:    data,
	}

	json.NewEncoder(w).Encode(response)
}

// sendErrorResponse sends error response
func (h *GroupsHandler) sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := GetGroupsResponse{
		Status:  "error",
		Message: message,
		Data:    []GroupInfo{},
	}

	json.NewEncoder(w).Encode(response)
}

