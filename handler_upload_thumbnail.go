package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	maxMemory := 10 << 20
	if err := r.ParseMultipartForm(int64(maxMemory)); err != nil {
		respondWithError(w, http.StatusInternalServerError, "error parsing multipart form", err)
		return
	}

	file, fileHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error getting file and file header", err)
		return
	}

	mediaType, _, err := mime.ParseMediaType(fileHeader.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error parsing media type", err)
		return
	}

	if mediaType != "image/png" && mediaType != "image/jpeg" {
		respondWithError(w, http.StatusBadRequest, "wrong file type", err)
		return
	}

	splitted := strings.Split(mediaType, "/")
	// imageData, err := io.ReadAll(file)
	// if err != nil {
	// 	respondWithError(w, http.StatusInternalServerError, "error getting image data", err)
	// 	return
	// }

	// imageDataString := base64.StdEncoding.EncodeToString(imageData)
	// imageDataURL := "data:" + mediaType + ";base64," + imageDataString

	dbVideo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "video not found", err)
		return
	}

	if dbVideo.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "not authorized", err)
		return
	}

	randSlice := make([]byte, 32)

	_, readErr := rand.Read(randSlice)
	if readErr != nil {
		respondWithError(w, http.StatusInternalServerError, "error creating random bytes", err)
		return
	}
	randString := base64.RawURLEncoding.EncodeToString(randSlice)

	imageFileName := randString + "." + splitted[1]

	fullPath := filepath.Join(cfg.assetsRoot, imageFileName)
	imageFile, err := os.Create(fullPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error creating file", err)
	}
	io.Copy(imageFile, file)

	thumbnailURL := "http://localhost:8091/assets/" + imageFileName
	dbVideo.ThumbnailURL = &thumbnailURL
	dbVideo.UpdatedAt = time.Now()
	if err := cfg.db.UpdateVideo(dbVideo); err != nil {
		respondWithError(w, http.StatusInternalServerError, "error updating video in db", err)
	}

	respondWithJSON(w, http.StatusOK, dbVideo)
}
