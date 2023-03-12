package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type BlogPost struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"`
	Title   string             `bson:"title"`
	Content string             `bson:"content"`
}

func getDB() (*mongo.Client, error) {
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return nil, err
	}

	err = client.Ping(context.Background(), readpref.Primary())
	if err != nil {
		return nil, err
	}

	return client, nil
}
func createBlogPost(c *gin.Context) {
	var blogPost BlogPost
	if err := c.ShouldBindJSON(&blogPost); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client, err := getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to database"})
		return
	}
	defer client.Disconnect(context.Background())

	collection := client.Database("blog").Collection("posts")
	result, err := collection.InsertOne(context.Background(), blogPost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert blog post"})
		return
	}

	id := result.InsertedID.(primitive.ObjectID)
	blogPost.ID = id
	c.JSON(http.StatusCreated, blogPost)
}

func getBlogPost(c *gin.Context) {
	id := c.Param("id")

	client, err := getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to database"})
		return
	}
	defer client.Disconnect(context.Background())

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var blogPost BlogPost
	collection := client.Database("blog").Collection("posts")
	err = collection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&blogPost)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Blog post not found"})
		return
	}

	c.JSON(http.StatusOK, blogPost)
}

func getBlogPosts(c *gin.Context) {
	client, err := getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to database"})
		return
	}
	defer client.Disconnect(context.Background())

	var blogPosts []BlogPost
	collection := client.Database("blog").Collection("posts")
	cur, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve blog posts"})
		return
	}
	defer cur.Close(context.Background())

	for cur.Next(context.Background()) {
		var blogPost BlogPost
		err := cur.Decode(&blogPost)
		if err != nil {
			log.Fatal(err)
		}
		blogPosts = append(blogPosts, blogPost)
	}

	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}

	c.JSON(http.StatusOK, blogPosts)
}

func updateBlogPost(c *gin.Context) {
	id := c.Param("id")
	var blogPost BlogPost
	if err := c.ShouldBindJSON(&blogPost); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client, err := getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to database"})
		return
	}
	defer client.Disconnect(context.Background())

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{"$set": bson.M{
		"title":   blogPost.Title,
		"content": blogPost.Content,
	}}

	collection := client.Database("blog").Collection("posts")
	result, err := collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update blog post"})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Blog post not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Blog post with ID %s updated", id)})
}

func deleteBlogPost(c *gin.Context) {
	id := c.Param("id")
	client, err := getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to database"})
		return
	}
	defer client.Disconnect(context.Background())

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	filter := bson.M{"_id": objectID}
	collection := client.Database("blog").Collection("posts")
	result, err := collection.DeleteOne(context.Background(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete blog post"})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Blog post not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Blog post with ID %s deleted", id)})
}

func main() {
	r := gin.Default()

	// Create blog post
	r.POST("/posts", createBlogPost)

	// Get blog post by ID
	r.GET("/posts/:id", getBlogPost)

	// Get all blog posts
	r.GET("/posts", getBlogPosts)

	// Update blog post by ID
	r.PUT("/posts/:id", updateBlogPost)

	// Delete blog post by ID
	r.DELETE("/posts/:id", deleteBlogPost)

	log.Fatal(r.Run(":8080"))
}
