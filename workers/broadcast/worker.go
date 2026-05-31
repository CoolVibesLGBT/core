package broadcast

import (
	"bytes"
	"context"
	app "core/infrastructure/bootstrap"
	"core/models"
	"core/models/utils"
	"time"

	"core/repositories"
	userservice "core/services/user"
	"core/workers"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func StartFetcher(dispatcher *workers.Dispatcher, a *app.App) {
	ticker := time.NewTicker(5 * time.Minute)

	// Başlangıçta 1 kez anında çalıştır
	dispatcher.SubmitEx(func() {
		fetchAndProcess(a)
	})

	go func() {
		for {
			<-ticker.C
			dispatcher.SubmitEx(func() {
				fetchAndProcess(a)
			})
		}
	}()
}

func fetchAndProcess(a *app.App) {
	log.Println("[BroadcastWorker] Fetching broadcasts...")

	if err := a.DB.Model(&models.User{}).Where("is_bot = ?", true).Updates(map[string]interface{}{"is_live": false, "is_online": true}).Error; err != nil {
		log.Printf("[BroadcastWorker] Error resetting IsLive and IsOnline for bots: %v", err)
	}

	type apiResult struct {
		Name string
		Body []byte
		Err  error
	}
	fetch := func(name, url, token string, headers map[string]string, ch chan apiResult) {
		payload := map[string]interface{}{
			"pageSize":  100,
			"gender":    "all",
			"latitude":  56.465587404589485,
			"longitude": 37.57010769460817,
			"more":      true,
			"score":     "0",
		}
		jsonData, _ := json.Marshal(payload)

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			ch <- apiResult{name, nil, err}
			return
		}

		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		req.Header.Set("Accept", "*/*")
		req.Header.Set("X-Parse-Application-Id", "sns-video")
		req.Header.Set("X-Parse-Session-Token", token)
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			ch <- apiResult{name, nil, err}
			return
		}
		defer func() { _ = resp.Body.Close() }()

		body, err := io.ReadAll(resp.Body)
		ch <- apiResult{name, body, err}
	}

	ch := make(chan apiResult, 2)

	// Fetch Growlr
	go fetch(
		"gdata",
		"https://api.gateway.growlr-live.com/video-api/growlr/functions/sns-video:getTrendingBroadcasts",
		"r:cf7d80043703b5729f3d463f813a2f38",
		map[string]string{
			"Host":                    "api.gateway.growlr-live.com",
			"X-Parse-Client-Key":      "com.initechapps.growlr",
			"X-Parse-Installation-Id": "98f4e8f2-21f9-4b1b-9564-7131f57709a3",
			"X-Parse-OS-Version":      "26.3 (23D127)",
			"Accept-Language":         "ru-RU,ru;q=0.9",
			"X-Parse-Client-Version":  "i1.19.6",
			"User-Agent":              "growlr/16.46.1.0 ( network=growlr; ) ios/26.3.0 ( iPhone; ) TMGCommon/8.23.3",
		},
		ch,
	)

	// Fetch Hornet
	go fetch(
		"hdata",
		"https://api.gateway.hornet-live.com/video-api/hornet/functions/sns-video:getTrendingBroadcasts",
		"r:82c7599c5e8f922d6db6791a26e2fcbc",
		map[string]string{
			"accept-language": "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7",
			"origin":          "https://api.gateway.hornet-live.com",
			"referer":         "https://api.gateway.hornet-live.com/web-live/search/trending/all",
			"x-user-agent":    "hornet/78.1.6 web/3.16.0 ( variant=small; )",
			"user-agent":      "Mozilla/5.0",
		},
		ch,
	)

	for i := 0; i < 2; i++ {
		res := <-ch
		if res.Err != nil {
			log.Printf("[BroadcastWorker] Fetch error for %s: %v", res.Name, res.Err)
			continue
		}
		processBroadcastData(a, res.Body, res.Name)
	}
}

func processBroadcastData(a *app.App, data []byte, provider string) {
	var resp struct {
		Result struct {
			Broadcasts []map[string]interface{} `json:"broadcasts"`
		} `json:"result"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		log.Printf("[BroadcastWorker] Unmarshal error: %v", err)
		return
	}

	repo := repositories.NewMediaRepository(a.DB, a.SnowFlakeNode)
	userRepo := repositories.NewUserRepository(a.DB, nil, a.SnowFlakeNode, nil, nil)
	userService := userservice.NewUserService(userRepo, nil, repo, nil, nil)

	for _, b := range resp.Result.Broadcasts {
		b["provider"] = provider

		userDetailsObj, ok := b["userDetails"].(map[string]interface{})
		if !ok {
			continue
		}

		networkUserIdStr := normalizeBroadcastUserID(userDetailsObj["networkUserId"])
		memberIdStr := normalizeBroadcastUserID(userDetailsObj["memberId"])
		lookupIDs := uniqueNonEmptyStrings(networkUserIdStr, memberIdStr)
		if len(lookupIDs) == 0 {
			continue
		}
		if networkUserIdStr == "" {
			networkUserIdStr = lookupIDs[0]
		}

		publicID, err := strconv.ParseInt(networkUserIdStr, 10, 64)
		if err != nil {
			continue
		}

		var user models.User

		defaultLang := "en"
		if langStr, ok := b["language"].(string); ok && langStr != "" {
			defaultLang = strings.ToLower(langStr)
		}

		err = a.DB.
			Where(`
				(
					broadcast_info->'userDetails'->>'networkUserId' IN ?
					OR broadcast_info->'userDetails'->>'memberId' IN ?
				)
			`, lookupIDs, lookupIDs).
			First(&user).Error
		if err == gorm.ErrRecordNotFound {
			streamDesc := getString(b["streamDescription"])
			if streamDesc == "" {
				streamDesc = getString(userDetailsObj["streamDescription"])
			}

			username := getString(userDetailsObj["displayName"])
			if username == "" {
				username = fmt.Sprintf("user_%d", publicID)
			}
			displayname := getString(userDetailsObj["firstName"])
			if displayname == "" {
				displayname = username
			}

			var bio *utils.LocalizedString
			if streamDesc != "" {
				bio = utils.MakeLocalizedString(defaultLang, streamDesc)
			}

			var dateOfBirth *time.Time
			birthDateRaw := b["birthDate"]
			if birthDateRaw == nil {
				birthDateRaw = userDetailsObj["birthDate"]
			}
			if bDateObj, ok := birthDateRaw.(map[string]interface{}); ok {
				if isoStr, ok := bDateObj["iso"].(string); ok {
					if t, err := time.Parse(time.RFC3339, isoStr); err == nil {
						dateOfBirth = &t
					}
				}
			}

			userToCreate := &models.User{
				Domain:          models.CoolVibes,
				UserName:        username,
				DisplayName:     displayname,
				DefaultLanguage: defaultLang,
				Bio:             bio,
				DateOfBirth:     dateOfBirth,
			}

			createdUser, err := userService.CreateBotUser(context.Background(), userToCreate)
			if err != nil {
				log.Printf("[BroadcastWorker] Error creating user %s: %v", networkUserIdStr, err)
				continue
			}
			user = *createdUser
			log.Printf("[BroadcastWorker] Created new user: %s", networkUserIdStr)

			// Download profile pic
			profilePic, ok := userDetailsObj["profilePic"].(map[string]interface{})
			if ok {
				largeUrl := getString(profilePic["large"])
				if largeUrl != "" && largeUrl != "null" && strings.HasPrefix(largeUrl, "http") {
					_, err := userService.UpdateAvatarFromURL(context.Background(), largeUrl, &user)
					if err != nil {
						log.Printf("[BroadcastWorker] Error updating avatar from url for user %s (%s): %v", user.ID, largeUrl, err)
					}
				}
			}
		} else if err != nil {
			log.Printf("[BroadcastWorker] Error querying user %s: %v", networkUserIdStr, err)
			continue
		} else {
			// Existing user: Update IsLive and check if avatar is missing
			user.IsLive = true
			user.IsOnline = true

			if user.AvatarID == nil {
				profilePic, ok := userDetailsObj["profilePic"].(map[string]interface{})
				if ok {
					largeUrl := getString(profilePic["large"])
					if largeUrl != "" && largeUrl != "null" && strings.HasPrefix(largeUrl, "http") {
						_, err := userService.UpdateAvatarFromURL(context.Background(), largeUrl, &user)
						if err != nil {
							log.Printf("[BroadcastWorker] Error updating missing avatar for existing user %s (%s): %v", user.ID, largeUrl, err)
						}
					}
				}
			}
		}

		// Update BroadcastInfo and IsLive in any case (newly created or existing)
		bBytes, _ := json.Marshal(b)
		user.BroadcastInfo = datatypes.JSON(bBytes)
		user.IsLive = true
		user.IsOnline = true

		if err := a.DB.Model(&user).Updates(map[string]interface{}{
			"broadcast_info": user.BroadcastInfo,
			"is_live":        true,
			"is_online":      true,
		}).Error; err != nil {
			log.Printf("[BroadcastWorker] Error updating BroadcastInfo, IsLive, and IsOnline for user %s: %v", networkUserIdStr, err)
		}
	}
}

func getString(val interface{}) string {
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

func normalizeBroadcastUserID(val interface{}) string {
	switch v := val.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	default:
		return ""
	}
}

func uniqueNonEmptyStrings(values ...string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}
