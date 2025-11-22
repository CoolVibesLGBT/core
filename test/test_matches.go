package test

import (
	"context"
	"coolvibes/faker"
	"coolvibes/helpers"
	"coolvibes/repositories"
	"coolvibes/services/socket"
	"coolvibes/types"
	"fmt"

	"gorm.io/gorm"
)

func testMatches(db *gorm.DB, snowFlakeNode *helpers.Node, socketService *socket.SocketService) {
	fromUser := faker.CreateUser(db, snowFlakeNode)
	toUser := faker.CreateUser(db, snowFlakeNode)

	fmt.Println("FromUser", fromUser.ID)
	fmt.Println("ToUser", toUser.ID)

	engagementRepo := repositories.NewEngagementRepository(db)
	matchesRepo := repositories.NewMatchesRepository(db, engagementRepo)

	isFromMatched, _ := matchesRepo.RecordView(context.Background(), fromUser.ID, toUser.ID, types.ReactionLike)
	isToMatched, _ := matchesRepo.RecordView(context.Background(), toUser.ID, fromUser.ID, types.ReactionLike)

	isMatched, _ := matchesRepo.IsMatched(context.Background(), fromUser.ID, toUser.ID)

	fmt.Println("User 1", isFromMatched)
	fmt.Println("User 2", isToMatched)
	fmt.Println("User1 and User2", isMatched)

	fmt.Println("Starting user creation loop")

	for i := 0; i < 10; i++ {
		fmt.Println("Iteration:", i)
		testUser := faker.CreateUser(db, snowFlakeNode)

		fmt.Println("Created user:", testUser.ID)

		matched, err := matchesRepo.RecordView(context.Background(), fromUser.ID, testUser.ID, types.ReactionLike)
		if err != nil {
			fmt.Println("Error recording view:", err)
			continue
		}
		fmt.Println("RecordView result for user", testUser.ID, "match:", matched)
	}

	fmt.Println("User creation loop ended")

	likes, _ := matchesRepo.GetLikesAfter(context.Background(), fromUser.ID, nil, 20)
	fmt.Println("Total Likes", len(likes))

}

func testMatchesDetails(db *gorm.DB, snowFlakeNode *helpers.Node) {

	fromUser := faker.CreateUser(db, snowFlakeNode)
	engagementRepo := repositories.NewEngagementRepository(db)
	matchesRepo := repositories.NewMatchesRepository(db, engagementRepo)

	for i := 0; i < 5; i++ {
		fmt.Println("Iteration:", i)
		testUser := faker.CreateUser(db, snowFlakeNode)

		fmt.Println("Created user:", testUser.ID)

		matchedFirst, err := matchesRepo.RecordView(context.Background(), fromUser.ID, testUser.ID, types.ReactionLike)
		if err != nil {
			fmt.Println("Error recording view:", err)
			continue
		}

		matched, err := matchesRepo.RecordView(context.Background(), testUser.ID, fromUser.ID, types.ReactionLike)
		if err != nil {
			fmt.Println("Error recording view:", err)
			continue
		}
		fmt.Println("RecordView result for user", testUser.ID, "match:", matchedFirst, matched)
	}
	likes, _ := matchesRepo.GetLikesAfter(context.Background(), fromUser.ID, nil, 20)
	fmt.Println("Total Likes", len(likes))

}

func StartTest(db *gorm.DB, snowFlakeNode *helpers.Node) {
	testMatchesDetails(db, snowFlakeNode)
}
