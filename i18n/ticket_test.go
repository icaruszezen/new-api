package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// 工单提示直接展示给用户：每个语言文件都要有自己的译文，模板参数必须能渲染。
func TestTicketMessagesTranslatedInEveryLocale(t *testing.T) {
	keys := []string{
		MsgTicketNotFound, MsgTicketSubjectRequired, MsgTicketSubjectTooLong, MsgTicketContentRequired,
		MsgTicketContentTooLong, MsgTicketClosed, MsgTicketTooManyOpen, MsgTicketTooManyImages,
		MsgTicketImageTooLarge, MsgTicketImageInvalidType, MsgTicketImageInvalid, MsgTicketImageDimensionsTooLarge,
		MsgTicketDailyImageLimit, MsgTicketMessageLimit, MsgTicketAttachmentLimit, MsgTicketRequestTooLarge,
		MsgTicketInvalidStatus, MsgTicketInvalidForm, MsgTicketAttachmentNotFound, MsgTicketImageTooComplex,
		MsgTicketImageBusy, MsgTicketRequestInProgress,
	}
	for _, file := range []string{"locales/en.yaml", "locales/zh-CN.yaml", "locales/zh-TW.yaml"} {
		raw, err := localeFS.ReadFile(file)
		require.NoError(t, err)
		var messages map[string]string
		require.NoError(t, yaml.Unmarshal(raw, &messages))
		for _, key := range keys {
			assert.NotEmpty(t, messages[key], "%s is missing %s", file, key)
		}
	}

	require.NoError(t, Init())
	data := map[string]any{"Max": 5, "MaxMB": 8, "MaxSide": 12000, "MaxMegapixels": 25, "MaxPixelsWan": 2500}
	for _, lang := range []string{LangEn, LangZhCN, LangZhTW} {
		for _, key := range keys {
			message := Translate(lang, key, data)
			assert.NotEqual(t, key, message, "%s %s", lang, key)
			assert.NotContains(t, message, "<no value>", "%s %s uses an unknown template field", lang, key)
		}
	}
	assert.Equal(t, "You can have at most 5 open tickets. Wait for an existing ticket to be resolved or close one first",
		Translate(LangEn, MsgTicketTooManyOpen, data))
	// 中文按“万像素”表达像素总数
	assert.Equal(t, "图片尺寸过大：每边不能超过 12000 像素，总像素不能超过 2500 万像素",
		Translate(LangZhCN, MsgTicketImageDimensionsTooLarge, data))
	assert.Equal(t, "圖片尺寸過大：每邊不能超過 12000 像素，總像素不能超過 2500 萬像素",
		Translate(LangZhTW, MsgTicketImageDimensionsTooLarge, data))
}
