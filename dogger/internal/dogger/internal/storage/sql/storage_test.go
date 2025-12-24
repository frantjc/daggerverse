package sql_test

import (
	"database/sql"
	"testing"

	"github.com/frantjc/daggerverse/dogger/internal/dogger/internal/storage"
	sqlstorage "github.com/frantjc/daggerverse/dogger/internal/dogger/internal/storage/sql"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

type Storage interface {
	storage.ContainerStore
	storage.ImageStore
}

func setupTestStorage(t *testing.T) Storage {
	t.Helper()

	db := setupTestDB(t)

	// Create a storage instance with SQLite-compatible schema
	s, err := sqlstorage.New(db)
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}

	return s
}

func TestCreateAndGetContainer(t *testing.T) {
	s := setupTestStorage(t)
	ctx := t.Context()

	// Create a container
	container := &storage.Container{}
	containerName := "test-container"

	err := s.CreateContainer(ctx, containerName, container)
	if err != nil {
		t.Fatalf("create container: %v", err)
	}

	// Retrieve the container
	retrievedContainer, err := s.GetContainer(ctx, containerName)
	if err != nil {
		t.Fatalf("get container: %v", err)
	}

	if retrievedContainer == nil {
		t.Fatal("container is nil")
	}
}

func TestCreateAndGetImage(t *testing.T) {
	s := setupTestStorage(t)

	ctx := t.Context()

	// Create an image
	image := &storage.Image{}
	imageTag := "test-image:latest"

	err := s.CreateImage(ctx, imageTag, image)
	if err != nil {
		t.Fatalf("create image: %v", err)
	}

	// Retrieve the image
	retrievedImage, err := s.GetImage(ctx, imageTag)
	if err != nil {
		t.Fatalf("get image: %v", err)
	}

	if retrievedImage == nil {
		t.Fatal("image is nil")
	}
}

func TestNameContainer(t *testing.T) {
	s := setupTestStorage(t)

	ctx := t.Context()

	// Create a container
	container := &storage.Container{}
	initialName := "initial-name"

	err := s.CreateContainer(ctx, initialName, container)
	if err != nil {
		t.Fatalf("create container: %v", err)
	}

	// Add another name to the container
	additionalName := "additional-name"
	err = s.NameContainer(ctx, initialName, additionalName)
	if err != nil {
		t.Fatalf("name container: %v", err)
	}

	// Verify we can retrieve the container by both names
	container1, err := s.GetContainer(ctx, initialName)
	if err != nil {
		t.Fatalf("get container by initial name: %v", err)
	}

	container2, err := s.GetContainer(ctx, additionalName)
	if err != nil {
		t.Fatalf("get container by additional name: %v", err)
	}

	if container1 == nil || container2 == nil {
		t.Fatal("a container is nil")
	}
}

func TestTagImage(t *testing.T) {
	s := setupTestStorage(t)

	ctx := t.Context()

	// Create an image
	image := &storage.Image{}
	initialTag := "myapp:v1.0"

	err := s.CreateImage(ctx, initialTag, image)
	if err != nil {
		t.Fatalf("create image: %v", err)
	}

	// Add another tag to the image
	additionalTag := "myapp:latest"
	err = s.TagImage(ctx, initialTag, additionalTag)
	if err != nil {
		t.Fatalf("tag image: %v", err)
	}

	// Verify we can retrieve the image by both tags
	image1, err := s.GetImage(ctx, initialTag)
	if err != nil {
		t.Fatalf("get image by initial tag: %v", err)
	}

	image2, err := s.GetImage(ctx, additionalTag)
	if err != nil {
		t.Fatalf("get image by additional tag: %v", err)
	}

	if image1 == nil || image2 == nil {
		t.Fatal("an image is nil")
	}
}

func TestDeleteContainer(t *testing.T) {
	s := setupTestStorage(t)

	ctx := t.Context()

	// Create a container
	container := &storage.Container{}
	containerName := "container-to-delete"

	err := s.CreateContainer(ctx, containerName, container)
	if err != nil {
		t.Fatalf("create container: %v", err)
	}

	// Add another name
	additionalName := "another-name"
	err = s.NameContainer(ctx, containerName, additionalName)
	if err != nil {
		t.Fatalf("add additional name: %v", err)
	}

	// Delete the container
	err = s.DeleteContainer(ctx, containerName)
	if err != nil {
		t.Fatalf("delete container: %v", err)
	}

	// Verify the container is gone (both names should fail)
	_, err = s.GetContainer(ctx, containerName)
	if err == nil {
		t.Fatal("Expected error when getting deleted container by original name, but got none")
	}

	_, err = s.GetContainer(ctx, additionalName)
	if err == nil {
		t.Fatal("Expected error when getting deleted container by additional name, but got none")
	}
}

func TestDeleteImage(t *testing.T) {
	s := setupTestStorage(t)

	ctx := t.Context()

	// Create an image
	image := &storage.Image{}
	imageTag := "image-to-delete:v1"

	err := s.CreateImage(ctx, imageTag, image)
	if err != nil {
		t.Fatalf("create image: %v", err)
	}

	// Add another tag
	additionalTag := "image-to-delete:latest"
	err = s.TagImage(ctx, imageTag, additionalTag)
	if err != nil {
		t.Fatalf("add additional tag: %v", err)
	}

	// Delete the image
	err = s.DeleteImage(ctx, imageTag)
	if err != nil {
		t.Fatalf("delete image: %v", err)
	}

	// Verify the image is gone (both tags should fail)
	_, err = s.GetImage(ctx, imageTag)
	if err == nil {
		t.Fatal("Expected error when getting deleted image by original tag, but got none")
	}

	_, err = s.GetImage(ctx, additionalTag)
	if err == nil {
		t.Fatal("Expected error when getting deleted image by additional tag, but got none")
	}
}

func TestGetNonExistentContainer(t *testing.T) {
	s := setupTestStorage(t)

	ctx := t.Context()

	// Try to get a container that doesn't exist
	_, err := s.GetContainer(ctx, "non-existent")
	if err == nil {
		t.Fatal("Expected error when getting non-existent container, but got none")
	}
}

func TestGetNonExistentImage(t *testing.T) {
	s := setupTestStorage(t)

	ctx := t.Context()

	// Try to get an image that doesn't exist
	_, err := s.GetImage(ctx, "non-existent:tag")
	if err == nil {
		t.Fatal("expect error when getting non-existent image")
	}
}

func TestDuplicateNames(t *testing.T) {
	s := setupTestStorage(t)

	ctx := t.Context()

	// Create first container
	container1 := &storage.Container{}
	name1 := "unique-name"

	err := s.CreateContainer(ctx, name1, container1)
	if err != nil {
		t.Fatalf("create first container: %v", err)
	}

	// Try to create second container with same name (should fail due to UNIQUE constraint)
	container2 := &storage.Container{}
	err = s.CreateContainer(ctx, name1, container2)
	if err == nil {
		t.Fatal("expect error when creating container with duplicate name")
	}
}

func TestDuplicateTags(t *testing.T) {
	s := setupTestStorage(t)

	ctx := t.Context()

	// Create first image
	image1 := &storage.Image{}
	tag1 := "unique-tag:v1"

	err := s.CreateImage(ctx, tag1, image1)
	if err != nil {
		t.Fatalf("create first image: %v", err)
	}

	// Try to create second image with same tag (should fail due to UNIQUE constraint)
	image2 := &storage.Image{}
	err = s.CreateImage(ctx, tag1, image2)
	if err == nil {
		t.Fatal("expect error when creating image with duplicate tag")
	}
}

func TestCascadeDelete(t *testing.T) {
	s := setupTestStorage(t)

	ctx := t.Context()

	// Create a container with multiple names
	container := &storage.Container{}
	initialName := "cascade-test"

	err := s.CreateContainer(ctx, initialName, container)
	if err != nil {
		t.Fatalf("create container: %v", err)
	}

	// Add multiple additional names
	names := []string{"name1", "name2", "name3"}
	for _, name := range names {
		err = s.NameContainer(ctx, initialName, name)
		if err != nil {
			t.Fatalf("add name %s: %v", name, err)
		}
	}

	// Delete the container
	err = s.DeleteContainer(ctx, initialName)
	if err != nil {
		t.Fatalf("delete container: %v", err)
	}

	// Verify all names are gone due to cascade delete
	for _, name := range names {
		_, err = s.GetContainer(ctx, name)
		if err == nil {
			t.Fatalf("expect error when getting container by name %s after delete", name)
		}
	}
}
