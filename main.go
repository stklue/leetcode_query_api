package main

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/machinebox/graphql"
)

type Problem struct {
	Title      string   `json:"title"`
	TitleSlug  string   `json:"titleSlug"`
	Difficulty string   `json:"difficulty"`
	Content    string   `json:"content"`
	Tags       []string `json:"tags"`
}

type QuestionListResponse struct {
	ProblemsetQuestionList struct {
		TotalNum  int `json:"totalNum"`
		Questions []struct {
			Title      string `json:"title"`
			TitleSlug  string `json:"titleSlug"`
			Difficulty string `json:"difficulty"`
			Content    string `json:"content"`
			TopicTags  []struct {
				Name string `json:"name"`
				Slug string `json:"slug"`
			} `json:"topicTags"`
		} `json:"data"`
	} `json:"problemsetQuestionList"`
}

func fetchProblems(search string) ([]Problem, error) {
	client := graphql.NewClient("https://leetcode.com/graphql")

	// GraphQL query string
	query := `
query problemsetQuestionList($categorySlug: String, $limit: Int, $skip: Int, $filters: QuestionListFilterInput) {
  problemsetQuestionList: questionList(
    categorySlug: $categorySlug
    limit: $limit
    skip: $skip
    filters: $filters
  ) {
    totalNum
    data {
      title
      titleSlug
      difficulty
      content
      topicTags {
        name
        slug
      }
    }
  }
}`

	// GraphQL request
	req := graphql.NewRequest(query)
	req.Var("categorySlug", "all-code-essentials") // or "algorithms"
	req.Var("limit", 100)
	req.Var("skip", 0)
	req.Var("filters", map[string]interface{}{
		"searchKeywords": search,
	})

	var respData QuestionListResponse

	if err := client.Run(context.Background(), req, &respData); err != nil {
		return nil, err
	}

	questions := respData.ProblemsetQuestionList.Questions
	var problems []Problem
	for _, q := range questions {
		if strings.Contains(strings.ToLower(q.Title), strings.ToLower(search)) {
			var tags []string
			for _, tag := range q.TopicTags {
				tags = append(tags, tag.Name)
			}

			problems = append(problems, Problem{
				Title:      q.Title,
				TitleSlug:  q.TitleSlug,
				Difficulty: q.Difficulty,
				Content:    q.Content,
				Tags:       tags,
			})
		}
	}

	return problems, nil
}

func corsMiddleware() gin.HandlerFunc {
	// Define allowed origins as a comma-separated string
	originsString := "http://localhost:3000,https://bruteforce-app.vercel.app"
	var allowedOrigins []string
	if originsString != "" {
		// Split the originsString into individual origins and store them in allowedOrigins slice
		allowedOrigins = strings.Split(originsString, ",")
	}

	// Return the actual middleware handler function
	return func(c *gin.Context) {
		// Function to check if a given origin is allowed
		isOriginAllowed := func(origin string, allowedOrigins []string) bool {
			for _, allowedOrigin := range allowedOrigins {
				if origin == allowedOrigin {
					return true
				}
			}
			return false
		}

		// Get the Origin header from the request
		origin := c.Request.Header.Get("Origin")

		// Check if the origin is allowed
		if isOriginAllowed(origin, allowedOrigins) {
			// If the origin is allowed, set CORS headers in the response
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")
		}

		// Handle preflight OPTIONS requests by aborting with status 204
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		// Call the next handler
		c.Next()
	}
}

func main() {
	r := gin.Default()
	r.Use(corsMiddleware())
	r.GET("/search", func(c *gin.Context) {
		query := c.Query("q")
		if query == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing search query"})
			return
		}

		problems, err := fetchProblems(query)
		if err != nil {
			log.Println("Error fetching problems:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch problems"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"results": problems})
	})
	// client.Log = func(s string) { log.Println(s) }

	log.Println("Server running at http://localhost:8080")
	r.Run(":8080")
}
