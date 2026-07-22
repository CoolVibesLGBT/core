package test

import (
	"context"
	"core/application/types"
	"core/faker"
	"core/helpers"
	"core/infrastructure/repositories"
	"core/infrastructure/socket"
	"fmt"

	"gorm.io/gorm"
)

func testMatches(db *gorm.DB, snowFlakeNode *helpers.Node, socketService *socket.SocketService) {
	fromUser := faker.CreateUser(db, snowFlakeNode)
	toUser := faker.CreateUser(db, snowFlakeNode)

	fmt.Println("FromUser", fromUser.ID)
	fmt.Println("ToUser", toUser.ID)

	notificationRepo := repositories.NewNotificationRepository(db, snowFlakeNode)
	matchesRepo := repositories.NewMatchesRepository(db, notificationRepo)

	isFromMatched, _ := matchesRepo.RecordView(context.Background(), fromUser.ID, toUser.ID, types.ReactionLike)
	isToMatched, _ := matchesRepo.RecordView(context.Background(), toUser.ID, fromUser.ID, types.ReactionLike)

	fmt.Println("User 1", isFromMatched)
	fmt.Println("User 2", isToMatched)

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
	fmt.Println("Total Likes", len(likes.Users))

}

func testMatchesDetails(db *gorm.DB, snowFlakeNode *helpers.Node) {

	fromUser := faker.CreateUser(db, snowFlakeNode)
	notificationRepo := repositories.NewNotificationRepository(db, snowFlakeNode)
	matchesRepo := repositories.NewMatchesRepository(db, notificationRepo)

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
	fmt.Println("Total Likes", len(likes.Users))

}

func StartTest(db *gorm.DB, snowFlakeNode *helpers.Node) {
	testMatches(db, snowFlakeNode, nil)
	testMatchesDetails(db, snowFlakeNode)
}
