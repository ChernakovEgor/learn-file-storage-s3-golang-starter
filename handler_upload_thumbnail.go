package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not parse multipart", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not form file", err)
		return
	}
	mediaType := header.Header.Get("Content-Type")
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "incorrect mediaType", nil)
		return
	}
	log.Println("got", mediaType)

	// data, err := io.ReadAll(file)
	// if err != nil {
	// 	log.Fatalf("reading data from file: %v", err)
	// }

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		log.Fatalf("reading video data from db: %v", err)
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User is not the video owner", nil)
		return
	}

	// thumbnailEncoded := base64.StdEncoding.EncodeToString(data)
	// base64DataURL := fmt.Sprintf("data:%s;base64,%s", mediaType, thumbnailEncoded)

	b := make([]byte, 32)
	_, err = rand.Read(b)
	if err != nil {
		log.Printf("reading rand number: %v", err)
	}
	randomString := base64.RawURLEncoding.EncodeToString(b)
	thumbnailFileName := fmt.Sprintf("%s.%s", randomString, strings.TrimPrefix(mediaType, "image/"))
	thumbnailPath := filepath.Join(cfg.assetsRoot, thumbnailFileName)

	f, err := os.Create(thumbnailPath)
	if err != nil {
		log.Printf("error creating file: %v", err)
	}
	defer f.Close()

	_, err = io.Copy(f, file)
	if err != nil {
		log.Printf("error during copying: %v", err)
	}

	thumbnailURL := fmt.Sprintf("http://localhost:%s/%s", os.Getenv("PORT"), thumbnailPath)
	video.ThumbnailURL = &thumbnailURL
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		log.Fatalf("updating video: %v", err)
	}

	respondWithJSON(w, http.StatusOK, video)
}
