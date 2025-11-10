package main

import (
	"fmt"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
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

}
