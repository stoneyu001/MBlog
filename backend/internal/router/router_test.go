package router

import (
	"testing"

	"blog/internal/comments"
	"blog/internal/repository"
	"blog/internal/tracking"
)

func TestSetupRouterDoesNotExposeRuntimeContentMutationRoutes(t *testing.T) {
	trackingRepo := repository.NewTrackingRepo(nil)
	trackingService := tracking.NewTrackingService(trackingRepo)
	defer trackingService.Close()

	engine := SetupRouter(
		trackingService,
		tracking.NewAnalyticsService(trackingRepo),
		comments.NewCommentService(repository.NewCommentRepo(nil)),
	)

	routes := make(map[string]bool)
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = true
	}

	for _, route := range []string{
		"GET /api/files",
		"GET /api/files/*filename",
		"POST /api/files",
		"DELETE /api/files/*filename",
		"POST /api/upload",
		"POST /api/build",
		"GET /static/*filepath",
	} {
		if routes[route] {
			t.Fatalf("不可变部署不应暴露运行时内容接口: %s", route)
		}
	}

	for _, route := range []string{"GET /admin", "GET /analytics", "GET /api/analytics"} {
		if !routes[route] {
			t.Fatalf("管理与分析入口应继续保留: %s", route)
		}
	}
}
