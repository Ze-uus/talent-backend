package websocket

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Ze-uus/talent-backend/internal/store"
)

type SSEHandler struct {
	st       store.Store
	interval time.Duration
}

func NewSSEHandler(s store.Store, interval time.Duration) *SSEHandler {
	return &SSEHandler{st: s, interval: interval}
}

// ServeHTTP streams campaign metrics to a generic campaign viewer.
// Route: GET /stream/{viewer_token}
// Requires a valid viewer_session cookie set after viewer password check.
func (h *SSEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("viewer_token")
	viewer, err := h.st.GetViewerByToken(r.Context(), token)
	if err != nil || !viewer.Active {
		http.Error(w, `{"message":"not_found","success":false}`, http.StatusNotFound)
		return
	}
	cookie, err := r.Cookie("viewer_session")
	if err != nil || cookie.Value != viewer.ID {
		http.Error(w, `{"message":"unauthorized","success":false}`, http.StatusUnauthorized)
		return
	}
	h.streamCampaignMetrics(w, r, viewer.Campaign_id)
}

// ServeBrandContact streams aggregated brand metrics to a brand contact.
// Route: GET /brand/view/{viewer_token}/live
// Requires a valid brand_contact_session cookie set after contact access check.
func (h *SSEHandler) ServeBrandContact(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("viewer_token")
	contact, err := h.st.GetBrandContactByViewerToken(r.Context(), token)
	if err != nil || !contact.Token_active {
		http.Error(w, `{"message":"not_found","success":false}`, http.StatusNotFound)
		return
	}
	cookie, err := r.Cookie("brand_contact_session")
	if err != nil || cookie.Value != token {
		http.Error(w, `{"message":"unauthorized","success":false}`, http.StatusUnauthorized)
		return
	}
	h.streamBrandMetrics(w, r, contact.Brand_id)
}

func (h *SSEHandler) streamCampaignMetrics(w http.ResponseWriter, r *http.Request, campaign_id string) {
	h.setSSEHeaders(w)
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming_not_supported", http.StatusInternalServerError)
		return
	}
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			campaign, err := h.st.GetCampaignByID(r.Context(), campaign_id)
			if err != nil {
				return
			}
			total, _ := h.st.GetTotalConversions(r.Context(), campaign_id)
			if _, err := fmt.Fprintf(w, "data: {\"campaign_id\":%q,\"name\":%q,\"total_conversions\":%g,\"timestamp\":%q}\n\n",
				campaign_id, campaign.Name, total, time.Now().UTC().Format(time.RFC3339)); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (h *SSEHandler) streamBrandMetrics(w http.ResponseWriter, r *http.Request, brand_id string) {
	h.setSSEHeaders(w)
	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			campaigns, err := h.st.ListCampaigns(r.Context(), store.CampaignFilter{Brand_id: brand_id, Status: "active"})
			if err != nil || len(campaigns) == 0 {
				if _, err := fmt.Fprintf(w, "data: {\"status\":\"no_active_campaigns\"}\n\n"); err != nil {
					return
				}
				flusher.Flush()
				continue
			}
			c := campaigns[0]
			total, _ := h.st.GetTotalConversions(r.Context(), c.ID)
			if _, err := fmt.Fprintf(w, "data: {\"campaign_id\":%q,\"name\":%q,\"total_conversions\":%g,\"timestamp\":%q}\n\n",
				c.ID, c.Name, total, time.Now().UTC().Format(time.RFC3339)); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (h *SSEHandler) setSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
}
