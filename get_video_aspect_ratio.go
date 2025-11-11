package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
)

func getVideoAspectRatio(filepath string) (string, error) {
	type streams struct {
	}
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filepath)

	var cmdOutput bytes.Buffer
	cmd.Stdout = &cmdOutput
	if err := cmd.Run(); err != nil {
		log.Printf("error running cmd %s", err)
	}
	jsonOutput := ffprobeOutput{}

	json.Unmarshal(cmdOutput.Bytes(), &jsonOutput)
	if len(jsonOutput.Streams) <= 0 {
		log.Printf("error, invalid stream data")
		return "", fmt.Errorf("invalid stream data")
	}

	videoWidth := jsonOutput.Streams[0].Width
	videoHeight := jsonOutput.Streams[0].Height
	ratioTolerance := 0.5
	landscape := 16.0 / 9.0
	portrait := 9.0 / 16.0
	ratio := float64(videoWidth) / float64(videoHeight)

	if (ratio >= landscape-ratioTolerance) && (ratio <= landscape+ratioTolerance) {
		return "16:9", nil
	}
	if (ratio >= portrait-ratioTolerance) && (ratio <= portrait+ratioTolerance) {
		return "9:16", nil
	}
	return "other", nil
}

type ffprobeOutput struct {
	Streams []struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"streams"`
}
