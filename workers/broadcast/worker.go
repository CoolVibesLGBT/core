package broadcast

import (
	"context"
	"core/application/ports"
	"core/models"
	"core/models/utils"
	"sync"
	"time"

	"core/workers"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
)

const fetchInterval = 5 * time.Minute

type Fetcher struct {
	cancel context.CancelFunc
	done   chan struct{}
	once   sync.Once
	tasks  sync.WaitGroup
}

func StartFetcher(dispatcher *workers.Dispatcher, dependencies Dependencies) *Fetcher {
	return StartFetcherContext(context.Background(), dispatcher, dependencies)
}

func StartFetcherContext(parent context.Context, dispatcher *workers.Dispatcher, dependencies Dependencies) *Fetcher {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	fetcher := &Fetcher{cancel: cancel, done: make(chan struct{})}
	if dispatcher == nil {
		log.Printf("[BroadcastWorker] dispatcher is not configured")
		close(fetcher.done)
		return fetcher
	}

	submit := func() {
		fetcher.tasks.Add(1)
		dispatcher.SubmitEx(func() {
			defer fetcher.tasks.Done()
			fetchAndProcess(ctx, dependencies)
		})
	}

	// Başlangıçta 1 kez anında çalıştır
	submit()

	go func() {
		defer func() {
			fetcher.tasks.Wait()
			close(fetcher.done)
		}()
		ticker := time.NewTicker(fetchInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				submit()
			}
		}
	}()

	return fetcher
}

func (f *Fetcher) Stop() {
	if f != nil && f.cancel != nil {
		f.once.Do(f.cancel)
	}
}

func (f *Fetcher) Shutdown(ctx context.Context) error {
	if f == nil {
		return nil
	}
	f.Stop()
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-f.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func fetchAndProcess(ctx context.Context, dependencies Dependencies) {
	log.Println("[BroadcastWorker] Fetching broadcasts...")
	if err := dependencies.validateFetcher(); err != nil {
		log.Printf("[BroadcastWorker] %v", err)
		return
	}

	if err := dependencies.Repository.ResetBotBroadcastPresence(ctx); err != nil {
		log.Printf("[BroadcastWorker] Error resetting IsLive and IsOnline for bots: %v", err)
	}

	type apiResult struct {
		Name string
		Body []byte
		Err  error
	}
	providers := []ports.BroadcastProvider{
		ports.BroadcastProviderGrowlr,
		ports.BroadcastProviderHornet,
	}
	ch := make(chan apiResult, len(providers))
	query := ports.BroadcastTrendingQuery{
		PageSize:  100,
		Gender:    "all",
		Latitude:  56.465587404589485,
		Longitude: 37.57010769460817,
		More:      true,
		Score:     "0",
	}
	for _, provider := range providers {
		provider := provider
		go func() {
			body, err := dependencies.Gateway.FetchTrending(ctx, provider, query)
			ch <- apiResult{Name: string(provider), Body: body, Err: err}
		}()
	}

	for range providers {
		res := <-ch
		if res.Err != nil {
			log.Printf("[BroadcastWorker] Fetch error for %s: %v", res.Name, res.Err)
			continue
		}
		if err := processBroadcastData(ctx, dependencies, res.Body, res.Name); err != nil {
			log.Printf("[BroadcastWorker] Processing error for %s: %v", res.Name, err)
		}
	}
}

func processBroadcastData(ctx context.Context, dependencies Dependencies, data []byte, provider string) error {
	if err := dependencies.validate(); err != nil {
		return err
	}
	var resp struct {
		Result struct {
			Broadcasts []map[string]interface{} `json:"broadcasts"`
		} `json:"result"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("unmarshal broadcasts: %w", err)
	}

	for _, b := range resp.Result.Broadcasts {
		if err := ctx.Err(); err != nil {
			return err
		}
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

		defaultLang := "en"
		if langStr, ok := b["language"].(string); ok && langStr != "" {
			defaultLang = strings.ToLower(langStr)
		}

		foundUser, found, err := dependencies.Repository.FindBroadcastUser(ctx, lookupIDs)
		if err != nil {
			log.Printf("[BroadcastWorker] Error querying user %s: %v", networkUserIdStr, err)
			continue
		}
		var user models.User
		if !found {
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

			createdUser, err := dependencies.Users.CreateBotUser(ctx, userToCreate)
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
					_, err := dependencies.Users.UpdateAvatarFromURL(ctx, largeUrl, &user)
					if err != nil {
						log.Printf("[BroadcastWorker] Error updating avatar from url for user %s (%s): %v", user.ID, largeUrl, err)
					}
				}
			}
		} else {
			if foundUser == nil {
				log.Printf("[BroadcastWorker] repository returned an empty user for %s", networkUserIdStr)
				continue
			}
			user = *foundUser
			// Existing user: Update IsLive and check if avatar is missing
			user.IsLive = true
			user.IsOnline = true

			if user.AvatarID == nil {
				profilePic, ok := userDetailsObj["profilePic"].(map[string]interface{})
				if ok {
					largeUrl := getString(profilePic["large"])
					if largeUrl != "" && largeUrl != "null" && strings.HasPrefix(largeUrl, "http") {
						_, err := dependencies.Users.UpdateAvatarFromURL(ctx, largeUrl, &user)
						if err != nil {
							log.Printf("[BroadcastWorker] Error updating missing avatar for existing user %s (%s): %v", user.ID, largeUrl, err)
						}
					}
				}
			}
		}

		// Update BroadcastInfo and IsLive in any case (newly created or existing)
		bBytes, _ := json.Marshal(b)
		if err := dependencies.Repository.UpdateBroadcastState(ctx, user.ID, bBytes); err != nil {
			log.Printf("[BroadcastWorker] Error updating BroadcastInfo, IsLive, and IsOnline for user %s: %v", networkUserIdStr, err)
		}
	}
	return nil
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
