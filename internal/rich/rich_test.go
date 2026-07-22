package rich

import (
	"encoding/json"
	"errors"
	"testing"
)

// testCoverAlias используется в тестах валидации медиа-алиасов.
const testCoverAlias = "cover"

// mustJSON маршалит значение в JSON-строку для сравнений в тестах.
func mustJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(data)
}

func TestRichTextRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		json string
	}{
		{"plain string", `"hello"`},
		{"array of strings", `["a","b"]`},
		{"bold wrapper", `{"type":"bold","text":"bold"}`},
		{"custom emoji", `{"type":"custom_emoji","custom_emoji_id":"123","alternative_text":"🔥"}`},
		{"url link", `{"type":"url","text":"link","url":"https://example.com"}`},
		{"mention", `{"type":"mention","text":"@user","username":"user"}`},
		{"mathematical expression", `{"type":"mathematical_expression","expression":"a+b"}`},
		{"anchor", `{"type":"anchor","name":"section1"}`},
		{"nested array with bold", `[{"type":"bold","text":"x"},"y"]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var rt RichText
			if err := json.Unmarshal([]byte(tt.json), &rt); err != nil {
				t.Fatalf("unmarshal %s: %v", tt.json, err)
			}
			out := mustJSON(t, rt)
			if out != tt.json {
				t.Fatalf("round-trip mismatch:\nwant: %s\n got: %s", tt.json, out)
			}
		})
	}
}

func TestRichTextUnmarshalRejectsInvalidToken(t *testing.T) {
	t.Parallel()

	var rt RichText
	if err := json.Unmarshal([]byte(`42`), &rt); err == nil {
		t.Fatal("expected error for numeric richtext")
	}
}

func TestCustomEmojiPreservedThroughConvert(t *testing.T) {
	t.Parallel()

	// RichMessage с параграфом, содержащим custom emoji.
	src := `{"is_rtl":false,"blocks":[
		{"type":"paragraph","text":[
			"Hello ",
			{"type":"custom_emoji","custom_emoji_id":"5368324170671202286","alternative_text":"👍"},
			" world"
		]}
	]}`

	var in RichMessage
	if err := json.Unmarshal([]byte(src), &in); err != nil {
		t.Fatalf("unmarshal rich message: %v", err)
	}

	out, err := Convert(in)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if out.IsRTL {
		t.Fatal("is_rtl must be false")
	}
	if len(out.Blocks) != 1 || out.Blocks[0].Type != BlockParagraph {
		t.Fatalf("expected one paragraph block, got %#v", out.Blocks)
	}

	// Параграф должен содержать массив из трёх RichText-узлов.
	wantItems := mustJSON(t, out.Blocks[0].Text)
	if wantItems != `["Hello ",{"type":"custom_emoji","custom_emoji_id":"5368324170671202286","alternative_text":"👍"}," world"]` {
		t.Fatalf("custom emoji not preserved:\n%s", wantItems)
	}

	// Рекурсивная сериализация всей структуры должна давать валидный JSON.
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal output: %v", err)
	}
	if !contains(string(data), `"custom_emoji_id":"5368324170671202286"`) {
		t.Fatalf("serialized output lost custom emoji: %s", data)
	}
	if !contains(string(data), `"alternative_text":"👍"`) {
		t.Fatalf("serialized output lost alternative text: %s", data)
	}
}

func TestConvertPhotoBlockPicksLargestFileID(t *testing.T) {
	t.Parallel()

	src := `{"blocks":[{"type":"photo","photo":[
		{"file_id":"small","width":90,"height":90,"file_size":100},
		{"file_id":"large","width":1280,"height":720,"file_size":1000},
		{"file_id":"medium","width":640,"height":480,"file_size":500}
	]}]}`

	var in RichMessage
	if err := json.Unmarshal([]byte(src), &in); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	out, err := Convert(in)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if len(out.Blocks) != 1 || out.Blocks[0].Photo.Type != MediaTypePhoto {
		t.Fatalf("unexpected photo block: %#v", out.Blocks)
	}
	if got := out.Blocks[0].Photo.Media; got != "large" {
		t.Fatalf("photo media = %q, want large", got)
	}
}

func TestConvertMediaBlocks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		json        string
		wantType    string
		wantMedia   InputMedia
		mediaGetter func(InputRichBlock) InputMedia
	}{
		{
			name:        "video",
			json:        `{"blocks":[{"type":"video","video":{"file_id":"vid","width":100,"height":50,"duration":5}}]}`,
			wantType:    BlockVideo,
			wantMedia:   InputMedia{Type: MediaTypeVideo, Media: "vid"},
			mediaGetter: func(b InputRichBlock) InputMedia { return b.Video },
		},
		{
			name:        "animation",
			json:        `{"blocks":[{"type":"animation","animation":{"file_id":"gif","width":100,"height":50,"duration":3}}]}`,
			wantType:    BlockAnimation,
			wantMedia:   InputMedia{Type: MediaTypeAnimation, Media: "gif"},
			mediaGetter: func(b InputRichBlock) InputMedia { return b.Animation },
		},
		{
			name:        "audio",
			json:        `{"blocks":[{"type":"audio","audio":{"file_id":"aud","duration":120,"title":"Song"}}]}`,
			wantType:    BlockAudio,
			wantMedia:   InputMedia{Type: MediaTypeAudio, Media: "aud"},
			mediaGetter: func(b InputRichBlock) InputMedia { return b.Audio },
		},
		{
			name:        "voice note",
			json:        `{"blocks":[{"type":"voice_note","voice_note":{"file_id":"voice","duration":10}}]}`,
			wantType:    BlockVoiceNote,
			wantMedia:   InputMedia{Type: MediaTypeVoiceNote, Media: "voice"},
			mediaGetter: func(b InputRichBlock) InputMedia { return b.VoiceNote },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var in RichMessage
			if err := json.Unmarshal([]byte(tt.json), &in); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			out, err := Convert(in)
			if err != nil {
				t.Fatalf("convert: %v", err)
			}
			if len(out.Blocks) != 1 || out.Blocks[0].Type != tt.wantType {
				t.Fatalf("unexpected block: %#v", out.Blocks)
			}
			media := tt.mediaGetter(out.Blocks[0])
			if media.Type != tt.wantMedia.Type || media.Media != tt.wantMedia.Media {
				t.Fatalf("media = {type: %q, media: %q}, want %+v", media.Type, media.Media, tt.wantMedia)
			}
		})
	}
}

func TestConvertRecursiveMediaInCollage(t *testing.T) {
	t.Parallel()

	src := `{"blocks":[{"type":"collage","blocks":[
		{"type":"photo","photo":[{"file_id":"p1","width":10,"height":10,"file_size":5}]},
		{"type":"photo","photo":[{"file_id":"p2","width":20,"height":20,"file_size":6}]}
	]}]}`

	var in RichMessage
	if err := json.Unmarshal([]byte(src), &in); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := Convert(in)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if len(out.Blocks) != 1 || out.Blocks[0].Type != BlockCollage {
		t.Fatalf("expected collage: %#v", out.Blocks)
	}
	if len(out.Blocks[0].Blocks) != 2 {
		t.Fatalf("expected two nested photos, got %d", len(out.Blocks[0].Blocks))
	}
	if out.Blocks[0].Blocks[0].Photo.Media != "p1" || out.Blocks[0].Blocks[1].Photo.Media != "p2" {
		t.Fatalf("nested photo file ids wrong: %+v", out.Blocks[0].Blocks)
	}
}

func TestConvertRecursiveMediaInListAndDetails(t *testing.T) {
	t.Parallel()

	src := `{"blocks":[
		{"type":"list","items":[
			{"label":"1","blocks":[{"type":"photo","photo":[{"file_id":"list-photo","width":1,"height":1,"file_size":1}]}]}
		]},
		{"type":"details","summary":"More","blocks":[
			{"type":"blockquote","blocks":[
				{"type":"video","video":{"file_id":"details-video","width":1,"height":1,"duration":1}}
			]}
		]}
	]}`

	var in RichMessage
	if err := json.Unmarshal([]byte(src), &in); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := Convert(in)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if len(out.Blocks) != 2 {
		t.Fatalf("expected two top-level blocks, got %d", len(out.Blocks))
	}
	if out.Blocks[0].Items[0].Blocks[0].Photo.Media != "list-photo" {
		t.Fatalf("list photo media wrong: %+v", out.Blocks[0].Items)
	}
	nested := out.Blocks[1].Blocks[0].Blocks[0]
	if nested.Type != BlockVideo || nested.Video.Media != "details-video" {
		t.Fatalf("nested video wrong: %#v", nested)
	}
}

func TestConvertUnknownBlockReturnsError(t *testing.T) {
	t.Parallel()

	src := `{"blocks":[{"type":"totally_new_block","text":"x"}]}`
	var in RichMessage
	if err := json.Unmarshal([]byte(src), &in); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	_, err := Convert(in)
	var unsupported *UnsupportedBlockError
	if !errors.As(err, &unsupported) {
		t.Fatalf("expected UnsupportedBlockError, got %v", err)
	}
	if unsupported.Type != "totally_new_block" {
		t.Fatalf("unexpected unsupported type: %q", unsupported.Type)
	}
}

func TestConvertPhotoBlockMissingFileID(t *testing.T) {
	t.Parallel()

	src := `{"blocks":[{"type":"photo","photo":[{"file_id":"","width":1,"height":1}]}]}`
	var in RichMessage
	if err := json.Unmarshal([]byte(src), &in); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	_, err := Convert(in)
	var missing *MissingFileIDError
	if !errors.As(err, &missing) {
		t.Fatalf("expected MissingFileIDError, got %v", err)
	}
}

func TestInputRichMessageKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		m    InputRichMessage
		want MessageKind
	}{
		{"empty", InputRichMessage{}, KindEmpty},
		{"markdown", InputRichMessage{Markdown: "# x"}, KindMarkdown},
		{"html", InputRichMessage{HTML: "<b>x</b>"}, KindHTML},
		{"blocks", InputRichMessage{Blocks: []InputRichBlock{{Type: BlockParagraph}}}, KindBlocks},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.m.Kind(); got != tt.want {
				t.Fatalf("Kind() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateRejectsMultipleContentSources(t *testing.T) {
	t.Parallel()

	m := InputRichMessage{Markdown: "# x", Blocks: []InputRichBlock{{Type: BlockParagraph}}}
	if err := Validate(m); err == nil {
		t.Fatal("expected validation error for markdown+blocks")
	}
	// Ровно один источник должен проходить.
	if err := Validate(InputRichMessage{Markdown: "# x"}); err != nil {
		t.Fatalf("markdown-only should validate: %v", err)
	}
}

func TestValidateRejectsEmpty(t *testing.T) {
	t.Parallel()

	if err := Validate(InputRichMessage{}); err == nil {
		t.Fatal("expected validation error for empty message")
	}
}

func TestValidateMediaAliasCharset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		alias   string
		wantErr bool
	}{
		{"valid short", "cover", false},
		{"valid with underscores", "photo_ab12cd34", false},
		{"valid max length", stringOfLen(64), false},
		{"empty", "", true},
		{"too long", stringOfLen(65), true},
		{"bad charset", "bad alias!", true},
		{"bad charset space", "bad alias", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateAlias(tt.alias)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateAlias(%q) err = %v, wantErr %v", tt.alias, err, tt.wantErr)
			}
		})
	}
}

func TestValidateRejectsDuplicateMediaAlias(t *testing.T) {
	t.Parallel()

	m := InputRichMessage{
		Markdown: "![](tg://photo?id=cover)",
		Media: []InputRichMessageMedia{
			{ID: testCoverAlias, Media: NewPhotoMedia("a")},
			{ID: testCoverAlias, Media: NewPhotoMedia("b")},
		},
	}
	if err := Validate(m); err == nil {
		t.Fatal("expected duplicate alias error")
	}
}

func TestValidateRejectsUnsupportedInputMediaType(t *testing.T) {
	t.Parallel()

	m := InputRichMessage{
		Markdown: "![](tg://photo?id=cover)",
		Media:    []InputRichMessageMedia{{ID: testCoverAlias, Media: InputMedia{Type: "document", Media: "x"}}},
	}
	if err := Validate(m); err == nil {
		t.Fatal("expected unsupported media type error")
	}
}

func TestValidateRejectsTooManyBlocks(t *testing.T) {
	t.Parallel()

	blocks := make([]InputRichBlock, MaxBlocks+1)
	for i := range blocks {
		blocks[i] = InputRichBlock{Type: BlockParagraph, Text: RichText{String: "x"}}
	}
	m := InputRichMessage{Blocks: blocks}
	if err := Validate(m); err == nil {
		t.Fatal("expected too-many-blocks error")
	}
}

func TestAliasRegistryRegisterAndLookup(t *testing.T) {
	t.Parallel()

	reg := NewAliasRegistry()
	media := InputRichMessageMedia{ID: "", Media: NewPhotoMedia("file-id")}
	alias, err := reg.Register(42, media)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := ValidateAlias(alias); err != nil {
		t.Fatalf("generated alias invalid: %v", err)
	}

	got, ok := reg.Lookup(42, alias)
	if !ok {
		t.Fatal("registered alias not found")
	}
	if got.Media.Media != "file-id" {
		t.Fatalf("lookup media = %q, want file-id", got.Media.Media)
	}

	// Другой владелец не видит алиас.
	if _, ok := reg.Lookup(7, alias); ok {
		t.Fatal("alias leaked across owners")
	}
}

func TestResolveReferencesUnknownAliasErrors(t *testing.T) {
	t.Parallel()

	reg := NewAliasRegistry()
	_, _, err := reg.ResolveReferences(42, "![](tg://photo?id=missing)")
	if !errors.Is(err, ErrUnknownMediaAlias) {
		t.Fatalf("expected ErrUnknownMediaAlias, got %v", err)
	}
}

func TestResolveReferencesReturnsMedia(t *testing.T) {
	t.Parallel()

	reg := NewAliasRegistry()
	alias, _ := reg.Register(42, InputRichMessageMedia{Media: NewPhotoMedia("file-id")})

	media, refs, err := reg.ResolveReferences(42, "![](tg://photo?id="+alias+")")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(media) != 1 || media[0].Media.Media != "file-id" {
		t.Fatalf("unexpected media: %+v", media)
	}
	if len(refs) != 1 || refs[0].Alias != alias {
		t.Fatalf("unexpected refs: %+v", refs)
	}
}

func TestExtractMediaRefs(t *testing.T) {
	t.Parallel()

	markdown := "![](tg://photo?id=a) text ![](tg://video?id=b) more tg://audio?id=c"
	refs := ExtractMediaRefs(markdown)
	if len(refs) != 3 {
		t.Fatalf("expected 3 refs, got %d", len(refs))
	}
	wantSchemes := []string{"photo", "video", "audio"}
	wantAliases := []string{"a", "b", "c"}
	for i, ref := range refs {
		if ref.Scheme != wantSchemes[i] || ref.Alias != wantAliases[i] {
			t.Fatalf("ref %d = %+v, want scheme %s alias %s", i, ref, wantSchemes[i], wantAliases[i])
		}
	}
}

func TestMarkdownPlusMediaSerializesCorrectly(t *testing.T) {
	t.Parallel()

	m := InputRichMessage{
		Markdown: "# Заголовок\n\n![](tg://photo?id=cover)",
		Media: []InputRichMessageMedia{
			{ID: testCoverAlias, Media: NewPhotoMedia("AgACAgIAAxkBAA")},
		},
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(data)
	if !contains(s, `"markdown":"# Заголовок`) {
		t.Fatalf("markdown missing: %s", s)
	}
	if !contains(s, `"id":"cover"`) {
		t.Fatalf("media id missing: %s", s)
	}
	if !contains(s, `"media":"AgACAgIAAxkBAA"`) {
		t.Fatalf("file id missing: %s", s)
	}
	if !contains(s, `"type":"photo"`) {
		t.Fatalf("media type missing: %s", s)
	}
}

func TestParagraphBlockOmitsUnrelatedMediaFields(t *testing.T) {
	t.Parallel()

	b := InputRichBlock{Type: BlockParagraph, Text: RichText{String: "hi"}}
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(data)
	for _, field := range []string{`"photo"`, `"video"`, `"animation"`, `"audio"`, `"voice_note"`, `"cells"`, `"items"`} {
		if contains(s, field) {
			t.Fatalf("paragraph block must not include field %s: %s", field, s)
		}
	}
}

func TestPhotoBlockEmitsPhotoFieldOnly(t *testing.T) {
	t.Parallel()

	b := InputRichBlock{Type: BlockPhoto, Photo: NewPhotoMedia("file-id")}
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(data)
	if !contains(s, `"photo":{"type":"photo","media":"file-id"}`) {
		t.Fatalf("photo field missing or malformed: %s", s)
	}
	for _, field := range []string{`"video":`, `"animation":`, `"audio":`, `"voice_note":`} {
		if contains(s, field) {
			t.Fatalf("photo block must not include field %s: %s", field, s)
		}
	}
}

func TestConvertPreservesAllTextStyles(t *testing.T) {
	t.Parallel()

	src := `{"blocks":[{"type":"paragraph","text":[
		{"type":"bold","text":["bold"]},
		{"type":"italic","text":["italic"]},
		{"type":"underline","text":["underline"]},
		{"type":"strikethrough","text":["strike"]},
		{"type":"code","text":["code"]},
		{"type":"spoiler","text":["secret"]}
	]}]}`

	var in RichMessage
	if err := json.Unmarshal([]byte(src), &in); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := Convert(in)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	items := out.Blocks[0].Text.Items
	if len(items) != 6 {
		t.Fatalf("expected 6 styled nodes, got %d", len(items))
	}
	wantTypes := []string{TextBold, TextItalic, TextUnderline, TextStrikethrough, TextCode, TextSpoiler}
	for i, want := range wantTypes {
		if items[i].Node == nil || items[i].Node.Type != want {
			t.Fatalf("node %d type = %v, want %s", i, items[i].Node, want)
		}
	}
}

func TestConvertPreservesTableCaption(t *testing.T) {
	t.Parallel()

	src := `{"blocks":[{"type":"table","cells":[[
		{"text":"A","align":"left","valign":"middle"}
	]],"caption":"My table"}]}`

	var in RichMessage
	if err := json.Unmarshal([]byte(src), &in); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := Convert(in)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if out.Blocks[0].Type != BlockTable {
		t.Fatalf("expected table block")
	}
	if out.Blocks[0].TableCaption.String != "My table" {
		t.Fatalf("table caption = %q, want My table", out.Blocks[0].TableCaption.String)
	}
	if len(out.Blocks[0].Cells) != 1 || out.Blocks[0].Cells[0][0].Text.String != "A" {
		t.Fatalf("table cell wrong: %+v", out.Blocks[0].Cells)
	}
}

func TestConvertPreservesMediaCaption(t *testing.T) {
	t.Parallel()

	src := `{"blocks":[{"type":"photo","photo":[{"file_id":"f","width":1,"height":1,"file_size":1}],
		"caption":{"text":"cap","credit":"author"}}]}`

	var in RichMessage
	if err := json.Unmarshal([]byte(src), &in); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := Convert(in)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	caption := out.Blocks[0].Caption
	if caption == nil || caption.Text.String != "cap" || caption.Credit.String != "author" {
		t.Fatalf("caption not preserved: %+v", caption)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(sub) > 0 && indexOf(s, sub) >= 0))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func stringOfLen(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a'
	}
	return string(b)
}
