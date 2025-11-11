package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	uploadLimit := 1 << 30
	http.MaxBytesReader(w, r.Body, int64(uploadLimit))
	videoIdString := r.PathValue("videoID")
	videoId, err := uuid.Parse(videoIdString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userId, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading video", videoId, "by user", userId)

	dbVideoMetadata, err := cfg.db.GetVideo(videoId)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't find video", err)
		return
	}

	if dbVideoMetadata.UserID != userId {
		respondWithError(w, http.StatusUnauthorized, "Not authorized", err)
		return
	}

	videoFormFile, _, formErr := r.FormFile("video")
	if formErr != nil {
		respondWithError(w, http.StatusInternalServerError, "error getting formfil", err)
		return
	}
	defer videoFormFile.Close()

	mediaType, _, parseErr := mime.ParseMediaType("video/mp4")
	if parseErr != nil {
		respondWithError(w, http.StatusInternalServerError, "error parsing media type", err)
		return
	}
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "invalid video format", err)
		return
	}
	tempUploadFile, err := os.CreateTemp("", "tubely-upload*.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error creating temp file", err)
		return
	}
	defer os.Remove(tempUploadFile.Name())
	defer tempUploadFile.Close()
	_, copyErr := io.Copy(tempUploadFile, videoFormFile)
	if copyErr != nil {
		respondWithError(w, http.StatusInternalServerError, "error writing to temp file", err)
		return
	}
	tempUploadFile.Seek(0, io.SeekStart)
	tempFilePath := tempUploadFile.Name()
	processedFilePath, err := processVideoForFastStart(tempFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error setting fast start", err)
		return
	}
	aspectRatio, err := getVideoAspectRatio(processedFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error getting aspect ratio", err)
		return
	}

	var prefix string
	if aspectRatio == "16:9" {
		prefix = "landscape"
	} else if aspectRatio == "9:16" {
		prefix = "portrait"
	} else {
		prefix = "other"
	}

	randSlice := make([]byte, 32)
	_, readErr := rand.Read(randSlice)
	if readErr != nil {
		respondWithError(w, http.StatusInternalServerError, "error creating random bytes", err)
		return
	}
	randString := base64.RawURLEncoding.EncodeToString(randSlice)
	videoFileName := prefix + "/" + randString + ".mp4"

	processedFile, err := os.Open(processedFilePath)
	defer os.Remove(processedFile.Name())
	defer processedFile.Close()

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error opening processed file", err)
	}

	_, uploadErr := cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &videoFileName,
		Body:        processedFile,
		ContentType: &mediaType,
	})
	if uploadErr != nil {
		respondWithError(w, http.StatusInternalServerError, "error uploading to s3 bucket", err)
	}

	videoUrl := "https://" + cfg.s3Bucket + ".s3." + cfg.s3Region + ".amazonaws.com/" + videoFileName

	if err := cfg.db.UpdateVideo(database.Video{
		ID:                videoId,
		CreatedAt:         dbVideoMetadata.CreatedAt,
		UpdatedAt:         time.Now(),
		ThumbnailURL:      dbVideoMetadata.ThumbnailURL,
		VideoURL:          &videoUrl,
		CreateVideoParams: dbVideoMetadata.CreateVideoParams,
	}); err != nil {
		respondWithError(w, http.StatusInternalServerError, "error saving to db", err)
	}
}
