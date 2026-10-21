package handlers

import (
	"core/adapters/inbound/http/middleware"
	"core/application/ports"
	"core/application/usecases"
	"core/constants"
	domainuser "core/domain/user"
	domainwallet "core/domain/wallet"
	"core/utils"
	"errors"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"strconv"
	"strings"
)

func HandleBroadcastSession(service *usecases.BroadcastGiftService, action string) fiber.Handler {
	return func(c fiber.Ctx) error {
		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok || user == nil || user.ID == uuid.Nil {
			return financialFeatureError(c, ports.ErrBroadcastUnauthorized)
		}
		var result ports.BroadcastSessionView
		var err error
		switch action {
		case "start":
			result, err = service.Start(c.Context(), user.ID)
		case "fetch":
			owner, parseErr := strconv.ParseInt(requestField(c, "user_id"), 10, 64)
			if parseErr != nil || owner <= 0 {
				return financialFeatureError(c, ports.ErrBroadcastInvalidInput)
			}
			result, err = service.Find(c.Context(), user.ID, owner)
		case "heartbeat", "end":
			id, parseErr := uuid.Parse(requestField(c, "broadcast_id"))
			if parseErr != nil || id == uuid.Nil {
				return financialFeatureError(c, ports.ErrBroadcastInvalidInput)
			}
			result, err = service.Change(c.Context(), user.ID, id, action == "end")
		default:
			return financialFeatureError(c, ports.ErrBroadcastInvalidInput)
		}
		if err != nil {
			return financialFeatureError(c, err)
		}
		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{"broadcast": result})
	}
}
func HandleBroadcastGift(service *usecases.BroadcastGiftService) fiber.Handler {
	return func(c fiber.Ctx) error {
		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok || user == nil || user.ID == uuid.Nil {
			return financialFeatureError(c, ports.ErrBroadcastUnauthorized)
		}
		id, err := uuid.Parse(requestField(c, "broadcast_id"))
		if err != nil || id == uuid.Nil {
			return financialFeatureError(c, ports.ErrBroadcastInvalidInput)
		}
		amount, err := parseTipAmount(requestField(c, "amount"))
		if err != nil {
			return financialFeatureError(c, err)
		}
		key := strings.TrimSpace(c.Get("Idempotency-Key"))
		if key == "" {
			key = requestField(c, "idempotency_key")
		}
		result, err := service.Gift(c.Context(), user.ID, id, amount, key)
		if err != nil {
			return financialFeatureError(c, err)
		}
		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{"gift": result})
	}
}
func HandleReferralRewards(service *usecases.ReferralRewardService, queue bool) fiber.Handler {
	return func(c fiber.Ctx) error {
		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok || user == nil || user.ID == uuid.Nil {
			return financialFeatureError(c, ports.ErrBroadcastUnauthorized)
		}
		cursor, err := parseMatchListCursor(requestField(c, "cursor"))
		if err != nil {
			return financialFeatureError(c, domainuser.ErrReferralReviewInvalid)
		}
		limit := constants.DEFAULT_LIMIT
		if raw := requestField(c, "limit"); raw != "" {
			limit, err = strconv.Atoi(raw)
			if err != nil || limit < 1 {
				return financialFeatureError(c, domainuser.ErrReferralReviewInvalid)
			}
		}
		status := domainuser.ReferralRewardStatus(requestField(c, "status"))
		if queue && status == "" {
			status = domainuser.ReferralRewardPending
		}
		rows, next, err := service.List(c.Context(), moderatorPrincipal(user.ID, user.UserRole), queue, status, cursor, limit)
		if err != nil {
			return financialFeatureError(c, err)
		}
		_, token, err := encodeSocialCursorPair(nil, next)
		if err != nil {
			return financialFeatureError(c, err)
		}
		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{"rewards": rows, "cursor": token})
	}
}
func HandleReferralRewardReview(service *usecases.ReferralRewardService) fiber.Handler {
	return func(c fiber.Ctx) error {
		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok || user == nil || user.ID == uuid.Nil {
			return financialFeatureError(c, ports.ErrBroadcastUnauthorized)
		}
		id, err := uuid.Parse(requestField(c, "reward_id"))
		if err != nil || id == uuid.Nil {
			return financialFeatureError(c, domainuser.ErrReferralReviewInvalid)
		}
		result, err := service.Review(c.Context(), moderatorPrincipal(user.ID, user.UserRole), id, requestField(c, "decision"), requestField(c, "note"), requestField(c, "verification_reference"))
		if err != nil {
			return financialFeatureError(c, err)
		}
		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{"reward": result})
	}
}
func financialFeatureError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ports.ErrBroadcastUnauthorized):
		return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
	case errors.Is(err, domainuser.ErrReferralReviewForbidden), errors.Is(err, domainuser.ErrReferralUnavailable), errors.Is(err, domainwallet.ErrTransferUnavailable), errors.Is(err, domainwallet.ErrTransferBlocked), errors.Is(err, domainuser.ErrInteractionUnavailable), errors.Is(err, domainuser.ErrInteractionBlocked), errors.Is(err, ports.ErrBroadcastSessionForbidden):
		return utils.SendError(c, fiber.StatusForbidden, constants.ErrPermissionDenied)
	case errors.Is(err, domainuser.ErrReferralReviewConflict), errors.Is(err, domainuser.ErrReferralIdentityUsed), errors.Is(err, domainwallet.ErrIdempotencyConflict):
		return utils.SendErrorWithMessage(c, fiber.StatusConflict, constants.ErrInvalidAction, err.Error())
	case errors.Is(err, ports.ErrNotFound), errors.Is(err, ports.ErrBroadcastSessionUnavailable):
		return utils.SendError(c, fiber.StatusNotFound, constants.ErrInvalidAction)
	case errors.Is(err, domainuser.ErrReferralReviewInvalid), errors.Is(err, ports.ErrBroadcastInvalidInput), errors.Is(err, domainwallet.ErrSelfTransfer), errors.Is(err, domainwallet.ErrInsufficientFunds), errors.Is(err, domainwallet.ErrAmountOutOfRange), errors.Is(err, domainwallet.ErrInvalidAmount), errors.Is(err, domainwallet.ErrAmountBelowMinimum), errors.Is(err, domainwallet.ErrIdempotencyKeyRequired), errors.Is(err, domainwallet.ErrInvalidIdempotencyKey):
		return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
	default:
		return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrDatabaseError)
	}
}
