package media

import (
	"core/helpers"
	"core/repositories"
	"log"
	"time"

	"gorm.io/gorm"
)

const (
	processorWorkerCount = 2
	processorIdleDelay   = time.Second
	staleProcessingAfter = 10 * time.Minute
	staleSweepInterval   = time.Minute
)

func StartProcessor(db *gorm.DB, node *helpers.Node) {
	repo := repositories.NewMediaRepository(db, node)

	for i := 0; i < processorWorkerCount; i++ {
		go runWorker(repo)
	}

	go runStaleSweep(repo)
}

func runWorker(repo *repositories.MediaRepository) {
	for {
		item, err := repo.ClaimNextPendingMedia(time.Now())
		if err != nil {
			log.Printf("[MediaProcessor] claim failed: %v", err)
			time.Sleep(processorIdleDelay)
			continue
		}
		if item == nil {
			time.Sleep(processorIdleDelay)
			continue
		}

		if err := repo.ProcessClaimedMedia(item); err != nil {
			log.Printf("[MediaProcessor] processing media %s failed: %v", item.ID, err)
		}
	}
}

func runStaleSweep(repo *repositories.MediaRepository) {
	ticker := time.NewTicker(staleSweepInterval)
	defer ticker.Stop()

	for range ticker.C {
		rows, err := repo.RequeueStaleProcessing(staleProcessingAfter)
		if err != nil {
			log.Printf("[MediaProcessor] stale sweep failed: %v", err)
			continue
		}
		if rows > 0 {
			log.Printf("[MediaProcessor] re-queued %d stale media rows", rows)
		}
	}
}
