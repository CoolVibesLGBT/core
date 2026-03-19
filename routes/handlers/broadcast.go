package handlers

import (
	"bytes"
	"core/middleware"
	services "core/services/user"
	"core/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gofiber/fiber/v3"
)

type BroadcastHandler struct {
	service *services.UserService
}

func NewBroadcastHandler(service *services.UserService) *BroadcastHandler {
	return &BroadcastHandler{service: service}
}

func HandleFetchBroadcasts(s *services.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {

		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			//return utils.SendErrorWithMessage(c, fiber.StatusUnauthorized, constants.ErrUnauthorized, "User not authenticated")
		}

		sessionToken := "r:82c7599c5e8f922d6db6791a26e2fcbc"

		if user != nil {
			if user.Location != nil {
				fmt.Println("USER", user.Location.Latitude, user.Location.Longitude)

			}
		}

		payload := map[string]interface{}{
			"pageSize":  10000,
			"gender":    "all",
			"latitude":  00.00,
			"longitude": 00.00,
			"more":      true,
			"score":     "0",
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "Failed to encode request",
			})
		}

		req, err := http.NewRequest("POST",
			"https://api.gateway.hornet-live.com/video-api/hornet/functions/sns-video:getTrendingBroadcasts",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "Failed to create request",
			})
		}

		// Headers
		req.Header.Set("accept", "application/json")
		req.Header.Set("accept-language", "tr-TR,tr;q=0.9,en-US;q=0.8,en;q=0.7")
		req.Header.Set("cache-control", "no-cache")
		req.Header.Set("content-type", "application/json; charset=UTF-8")
		req.Header.Set("origin", "https://api.gateway.hornet-live.com")
		req.Header.Set("referer", "https://api.gateway.hornet-live.com/web-live/search/trending/all")
		req.Header.Set("x-parse-application-id", "sns-video")
		req.Header.Set("x-parse-session-token", sessionToken)
		req.Header.Set("x-user-agent", "hornet/78.1.6 web/3.16.0 ( variant=small; )")
		req.Header.Set("user-agent", "Mozilla/5.0")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "Request failed",
			})
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return c.Status(fiber.ErrBadGateway.Code).JSON(fiber.Map{
				"error": "Failed to read response",
			})
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusCreated, string(body), "Broadcasts fetched successfully")

	}
}

func HandleBroadcastsJoinRequest(s *services.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {

		user, _ := middleware.GetAuthenticatedUser(c)

		sessionToken := "r:d5e7e6ca3df85e67927fb96cf655324b"

		broadcastId := c.FormValue("broadcastId")
		streamClientId := c.FormValue("streamClientId")

		if broadcastId == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "broadcastId is required",
			})
		}

		if streamClientId == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "streamClientId is required",
			})
		}

		payload := map[string]interface{}{
			"broadcastId":    broadcastId,
			"streamClientId": streamClientId,
		}

		if user != nil {
			if user.Location != nil {
				fmt.Println("user", user.Location.Latitude, user.Location.Longitude)
			}
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "Failed to encode request",
			})
		}

		req, err := http.NewRequest("POST",
			"https://api.gateway.hornet-live.com/video-api/hornet/functions/sns-video:requestToGuestBroadcast",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "Failed to create request",
			})
		}

		// 🔥 Headers (curl’dan birebir)
		req.Header.Set("accept", "application/json")
		req.Header.Set("accept-language", "tr-TR,tr;q=0.9,en-US;q=0.8,en;q=0.7")
		req.Header.Set("cache-control", "no-cache")
		req.Header.Set("content-type", "application/json; charset=UTF-8")
		req.Header.Set("origin", "https://api.gateway.hornet-live.com")
		req.Header.Set("referer", "https://api.gateway.hornet-live.com/web-live/view/jNcAuTFcJf/trending")

		req.Header.Set("x-parse-application-id", "sns-video")
		req.Header.Set("x-parse-session-token", sessionToken)
		req.Header.Set("x-user-agent", "hornet/78.1.6 web/3.16.0 ( variant=small; )")
		req.Header.Set("user-agent", "Mozilla/5.0")

		// optional ama bazen gerekli oluyor
		req.Header.Set("newrelic", "eyJ2IjpbMCwxXSwiZCI6eyJ0eSI6IkJyb3dzZXIiLCJhYyI6IjE5MDcyNyIsImFwIjoiNTk0MzU4NDcyIiwiaWQiOiI0NTEzMjRiMDBjZGIxNWViIiwidHIiOiJlYjQxYmYyOGNmZTYxZjBmYWE5ZDI3NjE1YjkyYmQ3MyIsInRpIjoxNzczOTE5MTM1NDAwfX0=")
		req.Header.Set("x-newrelic-id", "VQ8HVlRUGwYDUlhVDwMGVw==")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "Request failed",
			})
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return c.Status(fiber.ErrBadGateway.Code).JSON(fiber.Map{
				"error": "Failed to read response",
			})
		}

		// debug
		fmt.Println("RAW:", string(body))

		return utils.SendSuccessWithMessage(c, fiber.StatusCreated, string(body), "Guest request sent")
	}
}
