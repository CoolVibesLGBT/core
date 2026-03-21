package handlers

import (
	"bytes"
	"core/constants"
	"core/middleware"
	services "core/services/user"
	"core/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

type BroadcastHandler struct {
	service *services.UserService
}

func NewBroadcastHandler(service *services.UserService) *BroadcastHandler {
	return &BroadcastHandler{service: service}
}

func HandleFetchBroadcastsH(s *services.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {

		user, _ := middleware.GetAuthenticatedUser(c)

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
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrDatabaseError, "Failed to encode request"+err.Error())

		}

		req, err := http.NewRequest("POST",
			"https://api.gateway.hornet-live.com/video-api/hornet/functions/sns-video:getTrendingBroadcasts",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrDatabaseError, "Failed to create request"+err.Error())
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
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrDatabaseError, "Request failed"+err.Error())
		}
		defer func() {
			if cerr := resp.Body.Close(); cerr != nil && err == nil {
				err = cerr
			}
		}()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrDatabaseError, "Failed to read response"+err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusCreated, string(body), "Broadcasts fetched successfully")

	}
}

func HandleFetchBroadcastsG(s *services.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {

		user, _ := middleware.GetAuthenticatedUser(c)

		sessionToken := "r:cf7d80043703b5729f3d463f813a2f38"
		fmt.Println("ses", sessionToken)
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
			return utils.SendErrorWithMessage(
				c,
				fiber.StatusBadRequest,
				constants.ErrDatabaseError,
				"Failed to encode request"+err.Error(),
			)
		}

		req, err := http.NewRequest(
			"POST",
			"https://api.gateway.growlr-live.com/video-api/growlr/functions/sns-video:getTrendingBroadcasts",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			return utils.SendErrorWithMessage(
				c,
				fiber.StatusBadRequest,
				constants.ErrDatabaseError,
				"Failed to create request"+err.Error(),
			)
		}

		req.Header.Set("Host", "api.gateway.growlr-live.com")
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		req.Header.Set("Accept", "*/*")
		req.Header.Set("X-Parse-Session-Token", sessionToken)
		req.Header.Set("X-Parse-Application-Id", "sns-video")
		req.Header.Set("X-Parse-Client-Key", "com.initechapps.growlr")
		req.Header.Set("X-Parse-Installation-Id", "98f4e8f2-21f9-4b1b-9564-7131f57709a3")
		req.Header.Set("X-Parse-OS-Version", "26.3 (23D127)")
		req.Header.Set("Accept-Language", "tr-TR,tr;q=0.9")
		req.Header.Set("X-Parse-Client-Version", "i1.19.6")
		req.Header.Set("User-Agent", "growlr/16.46.1.0 ( network=growlr; ) ios/26.3.0 ( iPhone; ) TMGCommon/8.23.3")
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("X-Parse-App-Build-Version", "16.46.1.0")
		req.Header.Set("X-Parse-App-Display-Version", "16.46.1")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return utils.SendErrorWithMessage(
				c,
				fiber.StatusBadRequest,
				constants.ErrDatabaseError,
				"Request failed"+err.Error(),
			)
		}
		defer func() {
			if cerr := resp.Body.Close(); cerr != nil && err == nil {
				err = cerr
			}
		}()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return utils.SendErrorWithMessage(
				c,
				fiber.StatusBadRequest,
				constants.ErrDatabaseError,
				"Failed to read response"+err.Error(),
			)
		}

		fmt.Println("body", string(body))
		return utils.SendSuccessWithMessage(
			c,
			fiber.StatusCreated,
			string(body),
			"Broadcasts fetched successfully",
		)
	}
}

func HandleFetchBroadcasts(s *services.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {

		type result struct {
			Name string
			Body string
			Err  error
		}

		fetch := func(name, url, token string, headers map[string]string, ch chan result) {
			payload := map[string]interface{}{
				"pageSize":  10000,
				"gender":    "all",
				"latitude":  0.0,
				"longitude": 0.0,
				"more":      true,
				"score":     "0",
			}

			jsonData, _ := json.Marshal(payload)

			req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
			if err != nil {
				ch <- result{name, "", err}
				return
			}

			// ortak header
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
			req.Header.Set("Accept", "*/*")
			req.Header.Set("X-Parse-Application-Id", "sns-video")
			req.Header.Set("X-Parse-Session-Token", token)

			// özel headerlar
			for k, v := range headers {
				req.Header.Set(k, v)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				ch <- result{name, "", err}
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				ch <- result{name, "", err}
				return
			}

			ch <- result{name, string(body), nil}
		}

		ch := make(chan result, 2)

		go fetch(
			"gdata",
			"https://api.gateway.growlr-live.com/video-api/growlr/functions/sns-video:getTrendingBroadcasts",
			"r:cf7d80043703b5729f3d463f813a2f38",
			map[string]string{
				"Host":                        "api.gateway.growlr-live.com",
				"X-Parse-Client-Key":          "com.initechapps.growlr",
				"X-Parse-Installation-Id":     "98f4e8f2-21f9-4b1b-9564-7131f57709a3",
				"X-Parse-OS-Version":          "26.3 (23D127)",
				"Accept-Language":             "tr-TR,tr;q=0.9",
				"X-Parse-Client-Version":      "i1.19.6",
				"User-Agent":                  "growlr/16.46.1.0 ( network=growlr; ) ios/26.3.0 ( iPhone; ) TMGCommon/8.23.3",
				"Connection":                  "keep-alive",
				"X-Parse-App-Build-Version":   "16.46.1.0",
				"X-Parse-App-Display-Version": "16.46.1",
			},
			ch,
		)

		go fetch(
			"hdata",
			"https://api.gateway.hornet-live.com/video-api/hornet/functions/sns-video:getTrendingBroadcasts",
			"r:82c7599c5e8f922d6db6791a26e2fcbc",
			map[string]string{
				"accept-language": "tr-TR,tr;q=0.9,en-US;q=0.8,en;q=0.7",
				"cache-control":   "no-cache",
				"origin":          "https://api.gateway.hornet-live.com",
				"referer":         "https://api.gateway.hornet-live.com/web-live/search/trending/all",
				"x-user-agent":    "hornet/78.1.6 web/3.16.0 ( variant=small; )",
				"user-agent":      "Mozilla/5.0",
			},
			ch,
		)

		// sonuçları topla
		results := make(map[string]interface{})

		for i := 0; i < 2; i++ {
			res := <-ch
			if res.Err != nil {
				results[res.Name] = res.Err.Error()
			} else {
				results[res.Name] = json.RawMessage(res.Body) // JSON olarak sakla
			}
		}

		return utils.SendSuccessWithMessage(
			c,
			fiber.StatusOK,
			results,
			"All broadcasts fetched",
		)
	}
}

func HandleCreateBroadcast(s *services.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {

		streamDescription := c.FormValue("streamDescription")

		sessionToken := "r:82c7599c5e8f922d6db6791a26e2fcbc"

		payload := map[string]interface{}{
			"streamDescription": streamDescription,
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrDatabaseError, "Failed to encode request"+err.Error())
		}

		req, err := http.NewRequest("POST",
			"https://api.gateway.hornet-live.com/video-api/hornet/functions/sns-video:createBroadcast",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrDatabaseError, "Failed to create request"+err.Error())
		}

		// headers
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

		req.Header.Set("newrelic", "eyJ2IjpbMCwxXSwiZCI6eyJ0eSI6IkJyb3dzZXIiLCJhYyI6IjE5MDcyNyIsImFwIjoiNTk0MzU4NDcyIiwiaWQiOiIzMWE4MTk5NmJmZmQwNTY3IiwidHIiOiJjYzFlODk2MDgyOGIwNDNjNDllZjhkNGE1MmVkMzEyNiIsInRpIjoxNzczOTI2NTQ3ODY5fX0=")
		req.Header.Set("x-newrelic-id", "VQ8HVlRUGwYDUlhVDwMGVw==")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrDatabaseError, "Request failed"+err.Error())
		}
		defer func() {
			if cerr := resp.Body.Close(); cerr != nil && err == nil {
				err = cerr
			}
		}()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrDatabaseError, "Failed to read response"+err.Error())
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusOK, string(body), "Broadcast created")
	}
}

func HandleViewBroadcast(s *services.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {

		broadcastId := c.FormValue("broadcastId")

		if broadcastId == "" {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "broadcastId is required")
		}

		sessionToken := "r:82c7599c5e8f922d6db6791a26e2fcbc"

		payload := map[string]interface{}{
			"broadcastId":   broadcastId,
			"source":        "trending",
			"viewBroadcast": true,
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadGateway, constants.ErrInvalidInput, "Failed to encode request"+err.Error())
		}

		req, err := http.NewRequest("POST",
			"https://api.gateway.hornet-live.com/video-api/hornet/functions/sns-video:viewBroadcast",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadGateway, constants.ErrInvalidInput, "Failed to create request"+err.Error())
		}

		// headers
		req.Header.Set("accept", "application/json")
		req.Header.Set("accept-language", "tr-TR,tr;q=0.9,en-US;q=0.8,en;q=0.7")
		req.Header.Set("cache-control", "no-cache")
		req.Header.Set("content-type", "application/json; charset=UTF-8")
		req.Header.Set("origin", "https://api.gateway.hornet-live.com")
		req.Header.Set("referer", "https://api.gateway.hornet-live.com/web-live/view/"+broadcastId+"/trending")

		req.Header.Set("x-parse-application-id", "sns-video")
		req.Header.Set("x-parse-session-token", sessionToken)
		req.Header.Set("x-user-agent", "hornet/78.1.6 web/3.16.0 ( variant=small; )")
		req.Header.Set("user-agent", "Mozilla/5.0")

		// optional
		req.Header.Set("newrelic", "eyJ2IjpbMCwxXSwiZCI6eyJ0eSI6IkJyb3dzZXIiLCJhYyI6IjE5MDcyNyIsImFwIjoiNTk0MzU4NDcyIiwiaWQiOiJlY2Y3NDdkZTdkYjU0NTRkIiwidHIiOiIyOGEwMWIzNmRiNTFlM2UxYjFmZmMyNmNhN2NhOWM0NSIsInRpIjoxNzczOTI1NjkxODIwfX0=")
		req.Header.Set("x-newrelic-id", "VQ8HVlRUGwYDUlhVDwMGVw==")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadGateway, constants.ErrInvalidInput, "Request failed :"+err.Error())
		}
		defer func() {
			if cerr := resp.Body.Close(); cerr != nil && err == nil {
				err = cerr
			}
		}()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadGateway, constants.ErrInvalidInput, "Failed to read response:"+err.Error())
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusOK, string(body), "Viewer registered")
	}
}

func HandleBroadcastsJoinRequest(s *services.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {

		user, _ := middleware.GetAuthenticatedUser(c)

		sessionToken := "r:82c7599c5e8f922d6db6791a26e2fcbc"

		broadcastId := c.FormValue("broadcastId")
		streamClientId := c.FormValue("streamClientId")

		if broadcastId == "" {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "broadcastId is required")

		}

		if streamClientId == "" {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "streamClientId is required")
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
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "Failed to encode request"+err.Error())
		}

		req, err := http.NewRequest("POST",
			"https://api.gateway.hornet-live.com/video-api/hornet/functions/sns-video:requestToGuestBroadcast",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "Failed to create request"+err.Error())
		}
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
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "Request failed"+err.Error())
		}
		defer func() {
			if cerr := resp.Body.Close(); cerr != nil && err == nil {
				err = cerr
			}
		}()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "Failed to read response"+err.Error())
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusCreated, string(body), "Guest request sent")
	}
}

func HandleLikeBroadcast(s *services.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {

		broadcastId := c.FormValue("broadcastId")
		viewerId := c.FormValue("viewerId")
		numLikesStr := c.FormValue("numLikes")

		if broadcastId == "" {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "broadcastId is required")
		}
		if viewerId == "" {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "viewerId is required")
		}
		if numLikesStr == "" {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "numLikes is required")
		}

		sessionToken := "r:82c7599c5e8f922d6db6791a26e2fcbc"

		numLikes, err := strconv.Atoi(numLikesStr)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "numLikes must be a number")
		}

		payload := map[string]interface{}{
			"broadcastId": broadcastId,
			"viewerId":    viewerId,
			"numLikes":    numLikes,
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrDatabaseError, "Failed to encode request"+err.Error())
		}

		req, err := http.NewRequest("POST",
			"https://api.gateway.hornet-live.com/video-api/hornet/functions/sns-video:likeBroadcast",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrDatabaseError, "Failed to create request"+err.Error())
		}

		req.Header.Set("accept", "application/json")
		req.Header.Set("accept-language", "tr-TR,tr;q=0.9,en-US;q=0.8,en;q=0.7")
		req.Header.Set("cache-control", "no-cache")
		req.Header.Set("content-type", "application/json; charset=UTF-8")

		req.Header.Set("x-parse-application-id", "sns-video")
		req.Header.Set("x-parse-session-token", sessionToken)
		req.Header.Set("x-user-agent", "hornet/78.1.6 web/3.16.0 ( variant=small; )")
		req.Header.Set("user-agent", "Mozilla/5.0")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrDatabaseError, "Request failed"+err.Error())
		}
		defer func() {
			if cerr := resp.Body.Close(); cerr != nil && err == nil {
				err = cerr
			}
		}()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrDatabaseError, "Failed to read response"+err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, string(body), "Like sent")
	}
}
