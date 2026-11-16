package index

import (
 "encoding/json"
 "fmt"
 "testing"
)

func TestBusinessProductImagesAllRequireSafety(t *testing.T) {
 images := make([]any,4)
 for i := range images { images[i] = map[string]any{"image": map[string]any{"$type":"blob", "ref":map[string]any{"$link":"bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"},"mimeType":"image/png","size":12},"alt":fmt.Sprint(i)} }
 raw,_ := json.Marshal(map[string]any{"products":[]any{map[string]any{"title":"Wool","uri":"https://shop.example/wool","images":images}}})
 requirements,err := extractBusinessProfileImages(raw)
 if err != nil || len(requirements)!=4 { t.Fatalf("requirements = %v, %v",requirements,err) }
 for i, requirement := range requirements { if requirement.slot != fmt.Sprintf("products.0.images.%d",i) || requirement.blob == nil { t.Fatalf("requirement %d = %v",i,requirement) } }
}
