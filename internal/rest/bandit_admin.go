package rest

import (
	"net/http"
	"strconv"

	"myGreenMarket/business/bandit"
	"myGreenMarket/domain"

	"github.com/labstack/echo/v4"
)

type BanditAdminHandler struct {
	cfgRepo     bandit.ConfigRepository
	segmentRepo bandit.SegmentRepository
}

func NewBanditAdminHandler(
	cfgRepo bandit.ConfigRepository,
	segmentRepo bandit.SegmentRepository,
) *BanditAdminHandler {
	return &BanditAdminHandler{
		cfgRepo:     cfgRepo,
		segmentRepo: segmentRepo,
	}
}

// GET /api/v1/admin/bandit/config?slot=home_row1&variant=0
func (h *BanditAdminHandler) GetConfig(c echo.Context) error {
	ctx := c.Request().Context()
	slot := c.QueryParam("slot")
	variantStr := c.QueryParam("variant")

	if slot == "" || variantStr == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "slot and variant are required",
		})
	}

	variant, err := strconv.Atoi(variantStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "invalid variant",
		})
	}

	cfg, ok, err := h.cfgRepo.GetConfig(ctx, slot, variant)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": err.Error(),
		})
	}
	if !ok {
		return c.JSON(http.StatusNotFound, echo.Map{
			"error": "config not found",
		})
	}

	return c.JSON(http.StatusOK, cfg)
}

// PUT /api/v1/admin/bandit/config
// body: BanditConfig JSON
func (h *BanditAdminHandler) UpsertConfig(c echo.Context) error {
	ctx := c.Request().Context()

	var body domain.BanditConfig
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "invalid body: " + err.Error(),
		})
	}
	if body.Slot == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "slot is required",
		})
	}

	if err := h.cfgRepo.UpsertConfig(ctx, body); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"status": "ok",
	})
}

// GET /api/v1/admin/bandit/segment?user_id=123
func (h *BanditAdminHandler) GetSegment(c echo.Context) error {
	ctx := c.Request().Context()
	userIDStr := c.QueryParam("user_id")
	if userIDStr == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "user_id is required",
		})
	}
	userID64, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "invalid user_id",
		})
	}
	userID := uint(userID64)

	seg, ok, err := h.segmentRepo.GetSegment(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": err.Error(),
		})
	}
	if !ok {
		return c.JSON(http.StatusNotFound, echo.Map{
			"error": "segment not found",
		})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"user_id": userID,
		"segment": seg,
	})
}

// PUT /api/v1/admin/bandit/segment
// body: { "user_id": 123, "segment": 1 }
type upsertSegmentRequest struct {
	UserID  uint `json:"user_id"`
	Segment int  `json:"segment"`
}

func (h *BanditAdminHandler) UpsertSegment(c echo.Context) error {
	ctx := c.Request().Context()

	var body upsertSegmentRequest
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "invalid body: " + err.Error(),
		})
	}
	if body.UserID == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "user_id is required",
		})
	}

	if err := h.segmentRepo.UpsertSegment(ctx, body.UserID, body.Segment); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"status": "ok",
	})
}
