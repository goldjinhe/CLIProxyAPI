package helps

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

const (
	HappyHorseT2VModel = "happyhorse-1.1-t2v"
	HappyHorseI2VModel = "happyhorse-1.1-i2v"
	HappyHorseR2VModel = "happyhorse-1.1-r2v"
)

// HappyHorseModelMode returns the native HappyHorse mode for a model name.
func HappyHorseModelMode(model string) string {
	model = strings.TrimSpace(model)
	if index := strings.LastIndex(model, "/"); index >= 0 {
		model = model[index+1:]
	}
	if index := strings.Index(model, "("); index > 0 {
		model = model[:index]
	}
	switch strings.ToLower(strings.TrimSpace(model)) {
	case HappyHorseT2VModel:
		return HappyHorseT2VModel
	case HappyHorseI2VModel:
		return HappyHorseI2VModel
	case HappyHorseR2VModel:
		return HappyHorseR2VModel
	default:
		return ""
	}
}

// IsHappyHorseModel reports whether model is one of the supported HappyHorse 1.1 modes.
func IsHappyHorseModel(model string) bool {
	return HappyHorseModelMode(model) != ""
}

// CanonicalHappyHorseModel removes an optional provider prefix from a HappyHorse model.
func CanonicalHappyHorseModel(model string) string {
	if mode := HappyHorseModelMode(model); mode != "" {
		return mode
	}
	return strings.TrimSpace(model)
}

type happyHorseMedia struct {
	kind string
	url  string
}

// BuildHappyHorseVideoPayload converts the proxy's OpenAI-facing video request
// into the native asynchronous DashScope request.
func BuildHappyHorseVideoPayload(payload []byte, model string) ([]byte, error) {
	if !json.Valid(payload) {
		return nil, fmt.Errorf("body must be valid JSON")
	}
	mode := HappyHorseModelMode(model)
	if mode == "" {
		return nil, fmt.Errorf("model %q is not a supported HappyHorse model", strings.TrimSpace(model))
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("decode HappyHorse request: %w", err)
	}
	var input map[string]json.RawMessage
	if value := raw["input"]; len(value) > 0 {
		if err := json.Unmarshal(value, &input); err != nil {
			return nil, fmt.Errorf("input must be an object")
		}
	}

	prompt := firstRawString(raw["prompt"])
	if prompt == "" {
		prompt = firstRawString(input["prompt"])
	}
	if (mode == HappyHorseT2VModel || mode == HappyHorseR2VModel) && prompt == "" {
		return nil, fmt.Errorf("prompt is required for %s", mode)
	}

	media := make([]happyHorseMedia, 0)
	collectHappyHorseMedia(raw["media"], "", &media)
	collectHappyHorseMedia(input["media"], "", &media)

	firstImage := firstRawURL(raw["image_url"])
	if firstImage == "" {
		firstImage = firstRawURL(raw["image"])
	}
	if firstImage == "" {
		firstImage = firstRawURL(raw["input_reference"])
	}
	if firstImage == "" {
		firstImage = firstRawURL(input["image_url"])
	}
	if firstImage != "" {
		media = append(media, happyHorseMedia{kind: "first_frame", url: firstImage})
	}
	collectHappyHorseURLArray(raw["reference_images"], "reference_image", &media)
	collectHappyHorseURLArray(raw["reference_image_urls"], "reference_image", &media)
	collectHappyHorseURLArray(input["reference_images"], "reference_image", &media)

	parameters, err := happyHorseParameters(raw)
	if err != nil {
		return nil, err
	}
	if mode == HappyHorseI2VModel {
		delete(parameters, "ratio")
	}

	nativeInput := map[string]any{}
	if prompt != "" {
		nativeInput["prompt"] = prompt
	}
	switch mode {
	case HappyHorseT2VModel:
		if len(media) > 0 {
			return nil, fmt.Errorf("%s does not accept image or reference inputs", mode)
		}
	case HappyHorseI2VModel:
		if len(media) != 1 || media[0].kind != "first_frame" {
			return nil, fmt.Errorf("%s requires exactly one first-frame image", mode)
		}
		if err := validateHappyHorseURL(media[0].url); err != nil {
			return nil, fmt.Errorf("first-frame image: %w", err)
		}
		nativeInput["media"] = []map[string]string{{"type": "first_frame", "url": media[0].url}}
	case HappyHorseR2VModel:
		if len(media) < 1 || len(media) > 9 {
			return nil, fmt.Errorf("%s requires between 1 and 9 reference images", mode)
		}
		nativeMedia := make([]map[string]string, 0, len(media))
		for _, item := range media {
			if item.kind != "reference_image" {
				return nil, fmt.Errorf("%s accepts reference images only", mode)
			}
			if err := validateHappyHorseURL(item.url); err != nil {
				return nil, fmt.Errorf("reference image: %w", err)
			}
			nativeMedia = append(nativeMedia, map[string]string{"type": "reference_image", "url": item.url})
		}
		nativeInput["media"] = nativeMedia
	}

	native := map[string]any{
		"model":      mode,
		"input":      nativeInput,
		"parameters": parameters,
	}
	encoded, err := json.Marshal(native)
	if err != nil {
		return nil, fmt.Errorf("encode HappyHorse request: %w", err)
	}
	return encoded, nil
}

func happyHorseParameters(raw map[string]json.RawMessage) (map[string]any, error) {
	parameters := make(map[string]any)
	if value := raw["parameters"]; len(value) > 0 {
		if err := json.Unmarshal(value, &parameters); err != nil {
			return nil, fmt.Errorf("parameters must be an object")
		}
	}

	if value := raw["resolution"]; len(value) > 0 {
		parameters["resolution"] = firstRawString(value)
	}
	if value := raw["ratio"]; len(value) > 0 {
		parameters["ratio"] = firstRawString(value)
	} else if value := raw["aspect_ratio"]; len(value) > 0 {
		parameters["ratio"] = firstRawString(value)
	}
	if value := raw["duration"]; len(value) > 0 {
		parameters["duration"] = valueAsNumber(value)
	} else if value := raw["seconds"]; len(value) > 0 {
		parameters["duration"] = valueAsNumber(value)
	}
	for _, key := range []string{"watermark", "seed", "prompt_extend"} {
		if value := raw[key]; len(value) > 0 {
			var item any
			if err := json.Unmarshal(value, &item); err != nil {
				return nil, fmt.Errorf("parameter %s is invalid", key)
			}
			parameters[key] = item
		}
	}

	if size := firstRawString(raw["size"]); size != "" {
		resolution, ratio, err := happyHorseSizeParameters(size)
		if err != nil {
			return nil, err
		}
		if _, exists := parameters["resolution"]; !exists {
			parameters["resolution"] = resolution
		}
		if _, exists := parameters["ratio"]; !exists {
			parameters["ratio"] = ratio
		}
	}

	resolution, err := normalizeHappyHorseResolution(parameters["resolution"])
	if err != nil {
		return nil, err
	}
	parameters["resolution"] = resolution
	ratio, err := normalizeHappyHorseRatio(parameters["ratio"])
	if err != nil {
		return nil, err
	}
	parameters["ratio"] = ratio
	duration, err := normalizeHappyHorseDuration(parameters["duration"])
	if err != nil {
		return nil, err
	}
	parameters["duration"] = duration
	return parameters, nil
}

func normalizeHappyHorseResolution(value any) (string, error) {
	resolution := strings.ToUpper(strings.TrimSpace(fmt.Sprint(value)))
	if resolution == "<NIL>" || resolution == "" {
		return "1080P", nil
	}
	resolution = strings.TrimSuffix(resolution, "P") + "P"
	switch resolution {
	case "480P", "720P", "1080P":
		return resolution, nil
	default:
		return "", fmt.Errorf("resolution must be one of 480P, 720P, or 1080P")
	}
}

func normalizeHappyHorseRatio(value any) (string, error) {
	ratio := strings.ToLower(strings.TrimSpace(fmt.Sprint(value)))
	if ratio == "<nil>" || ratio == "" {
		return "16:9", nil
	}
	switch ratio {
	case "landscape":
		return "16:9", nil
	case "portrait":
		return "9:16", nil
	case "square":
		return "1:1", nil
	}
	parts := strings.Split(ratio, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("ratio must use the form width:height")
	}
	for _, part := range parts {
		if value, err := strconv.Atoi(strings.TrimSpace(part)); err != nil || value <= 0 {
			return "", fmt.Errorf("ratio must use the form width:height")
		}
	}
	switch ratio {
	case "16:9", "9:16", "1:1", "4:3", "3:4", "4:5", "5:4", "9:21", "21:9":
		return ratio, nil
	default:
		return "", fmt.Errorf("ratio is not supported by HappyHorse")
	}
}

func normalizeHappyHorseDuration(value any) (int, error) {
	if value == nil {
		return 5, nil
	}
	var duration int
	switch item := value.(type) {
	case json.Number:
		parsed, err := strconv.Atoi(string(item))
		if err != nil {
			return 0, fmt.Errorf("duration must be an integer between 3 and 15")
		}
		duration = parsed
	case float64:
		if item != float64(int(item)) {
			return 0, fmt.Errorf("duration must be an integer between 3 and 15")
		}
		duration = int(item)
	case int:
		duration = item
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(item))
		if err != nil {
			return 0, fmt.Errorf("duration must be an integer between 3 and 15")
		}
		duration = parsed
	default:
		return 0, fmt.Errorf("duration must be an integer between 3 and 15")
	}
	if duration < 3 || duration > 15 {
		return 0, fmt.Errorf("duration must be between 3 and 15 seconds")
	}
	return duration, nil
}

func happyHorseSizeParameters(size string) (string, string, error) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(size)), "x")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("size must use the form widthxheight")
	}
	width, errWidth := strconv.Atoi(strings.TrimSpace(parts[0]))
	height, errHeight := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errWidth != nil || errHeight != nil || width <= 0 || height <= 0 {
		return "", "", fmt.Errorf("size must use the form widthxheight")
	}
	resolution := "480P"
	if width >= 1600 || height >= 1600 {
		resolution = "1080P"
	} else if width >= 1000 || height >= 1000 {
		resolution = "720P"
	}
	greatestCommonDivisor := func(a, b int) int {
		for b != 0 {
			a, b = b, a%b
		}
		return a
	}
	divisor := greatestCommonDivisor(width, height)
	return resolution, fmt.Sprintf("%d:%d", width/divisor, height/divisor), nil
}

func collectHappyHorseMedia(value json.RawMessage, defaultKind string, out *[]happyHorseMedia) {
	if len(value) == 0 || out == nil {
		return
	}
	var items []json.RawMessage
	if json.Unmarshal(value, &items) != nil {
		if itemURL := rawURL(value); itemURL != "" {
			*out = append(*out, happyHorseMedia{kind: defaultKind, url: itemURL})
		}
		return
	}
	for _, item := range items {
		kind := defaultKind
		var object map[string]json.RawMessage
		if json.Unmarshal(item, &object) == nil {
			kind = strings.ToLower(strings.TrimSpace(firstRawString(object["type"])))
			if kind == "" {
				kind = defaultKind
			}
		}
		if itemURL := rawURL(item); itemURL != "" {
			*out = append(*out, happyHorseMedia{kind: kind, url: itemURL})
		}
	}
}

func collectHappyHorseURLArray(value json.RawMessage, kind string, out *[]happyHorseMedia) {
	collectHappyHorseMedia(value, kind, out)
}

func rawURL(value json.RawMessage) string {
	if len(value) == 0 {
		return ""
	}
	if item := firstRawString(value); item != "" {
		return item
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(value, &object) != nil {
		return ""
	}
	if item := firstRawString(object["url"]); item != "" {
		return item
	}
	if item := firstRawString(object["image_url"]); item != "" {
		return item
	}
	if nested := object["image_url"]; len(nested) > 0 {
		return rawURL(nested)
	}
	return ""
}

func firstRawURL(value json.RawMessage) string {
	return rawURL(value)
}

func firstRawString(value json.RawMessage) string {
	if len(value) == 0 {
		return ""
	}
	var item string
	if json.Unmarshal(value, &item) == nil {
		return strings.TrimSpace(item)
	}
	return ""
}

func valueAsNumber(value json.RawMessage) any {
	if item := firstRawString(value); item != "" {
		if parsed, err := strconv.Atoi(item); err == nil {
			return parsed
		}
		return item
	}
	var item any
	if json.Unmarshal(value, &item) == nil {
		return item
	}
	return nil
}

func validateHappyHorseURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(strings.ToLower(raw), "data:image/") {
		lower := strings.ToLower(raw)
		comma := strings.IndexByte(lower, ',')
		if comma >= 0 && strings.HasSuffix(lower[:comma], ";base64") && strings.TrimSpace(raw[comma+1:]) != "" {
			switch strings.TrimSuffix(lower[:comma], ";base64") {
			case "data:image/jpeg", "data:image/jpg", "data:image/png", "data:image/webp":
				return nil
			}
		}
		return fmt.Errorf("data URL must contain a base64-encoded JPEG, PNG, or WEBP image")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed == nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("URL must be an absolute http or https URL")
	}
	return nil
}

// HappyHorseTaskID returns the task identifier from a native create or retrieve response.
func HappyHorseTaskID(payload []byte) string {
	for _, path := range []string{"output.task_id", "task_id", "request_id"} {
		if value := strings.TrimSpace(gjson.GetBytes(payload, path).String()); value != "" {
			return value
		}
	}
	return ""
}

// HappyHorseTaskStatus returns the native task status in uppercase.
func HappyHorseTaskStatus(payload []byte) string {
	for _, path := range []string{"output.task_status", "task_status", "status"} {
		if value := strings.ToUpper(strings.TrimSpace(gjson.GetBytes(payload, path).String())); value != "" {
			return value
		}
	}
	return ""
}

// HappyHorseVideoURL returns the native completed video URL.
func HappyHorseVideoURL(payload []byte) string {
	for _, path := range []string{"output.video_url", "video_url", "video.url"} {
		if value := strings.TrimSpace(gjson.GetBytes(payload, path).String()); value != "" {
			return value
		}
	}
	return ""
}
