package api

import "testing"

func TestParseRedditBackgroundsKeepsOnlyDirectImages(t *testing.T) {
	feed := `<feed>
		<entry>
			<title>One &amp; Two</title>
			<link href="https://reddit.example/post"/>
			<content type="html">&lt;a href="https://i.redd.it/scene.webp"&gt;image&lt;/a&gt;</content>
		</entry>
		<entry>
			<title>Gallery</title>
			<link href="https://reddit.example/gallery"/>
			<content type="html">&lt;a href="https://reddit.com/gallery/1"&gt;gallery&lt;/a&gt;</content>
		</entry>
	</feed>`
	images := parseRedditBackgrounds(feed)
	if len(images) != 1 {
		t.Fatalf("images = %#v", images)
	}
	if images[0].URL != "https://i.redd.it/scene.webp" ||
		images[0].Title != "One & Two" ||
		images[0].Source != "https://reddit.example/post" ||
		images[0].Subreddit != "eveporn" {
		t.Errorf("image = %#v", images[0])
	}
}

func TestRedditBackgroundPayloadUsesStableEmptyArray(t *testing.T) {
	payload := redditBackgroundPayload(nil)
	body := payload.Body.(map[string]any)
	if images, ok := body["images"].([]redditBackground); !ok || images == nil {
		t.Fatalf("images = %#v, want non-nil array", body["images"])
	}
}
