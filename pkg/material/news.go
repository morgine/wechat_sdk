package material

import (
	"github.com/morgine/wechat/pkg"
)

type News struct {
	Articles []*Article `json:"articles"`
}

type NewsArticles struct {
	NewsItem []*Article `json:"news_item"`
}

type Article struct {
	// 标题
	Title string `json:"title"`

	// 图文消息的封面图片素材id（必须是永久mediaID）
	ThumbMediaID string `json:"thumb_media_id"`

	// 作者
	Author string `json:"author"`

	// 图文消息的摘要，仅有单图文消息才有摘要，多图文此处为空。如果本字段为没有填写，则默认抓取正文前64个字。
	Digest string `json:"digest"`

	// 是否显示封面，0为false，即不显示，1为true，即显示
	ShowCoverPic int `json:"show_cover_pic"`

	// 图文消息的具体内容，支持HTML标签，必须少于2万字符，小于1M，且此处会去除JS,涉及图片url必须来源 '上传图文消息内的图片获取URL'接口获取。外部图片url将被过滤。
	Content string `json:"content"`

	// 图文消息的原文地址，即点击“阅读原文”后的URL
	ContentSourceUrl string `json:"content_source_url"`

	// 上传图文之后该图文会有一个 Url 地址
	Url string `json:"url"`

	// 是否打开评论，0不打开，1打开
	NeedOpenComment int `json:"need_open_comment"`

	// 是否粉丝才可评论，0所有人可评论，1粉丝才可评论
	OnlyFansCanComment int `json:"only_fans_can_comment"`
}

// 上传图文并获得图文信息
func UploadNews(news *News, token string) (mediaID string, err error) {
	ul := "https://api.weixin.qq.com/cgi-bin/material/add_news?access_token=" + token
	res := &struct {
		MediaID string `json:"media_id"`
	}{}
	err = pkg.PostSchema(pkg.KindJson, ul, news, &res)
	if err != nil {
		return "", err
	} else {
		return res.MediaID, nil
	}
}

type ContentImage struct {
	Url string `json:"url"`
}

func UploadNewsContentImage(fileName, token string, data []byte) (uri string, err error) {
	uri = "https://api.weixin.qq.com/cgi-bin/media/uploadimg?access_token=" + token
	res := &ContentImage{}
	err = pkg.UploadFile(uri, data, "media", fileName, nil, res)
	if err != nil {
		return "", err
	} else {
		return res.Url, nil
	}
}

func GetNewsArticles(mediaID, token string) (articles []*Article, err error) {
	uri := "https://api.weixin.qq.com/cgi-bin/material/get_material?access_token=" + token
	res := &NewsArticles{}
	err = pkg.PostSchema(pkg.KindJson, uri, map[string]string{"media_id": mediaID}, res)
	if err != nil {
		return nil, err
	} else {
		return res.NewsItem, nil
	}
}

type NewsList struct {
	TotalCount int       `json:"total_count"`
	ItemCount  int       `json:"item_count"`
	Item       []NewItem `json:"item"`
}

type NewItem struct {
	MediaID    string       `json:"media_id"`
	Content    NewsArticles `json:"content"`
	UpdateTime int64        `json:"update_time"`
}

// 获得图文列表
func GetNews(token string, limit, offset int) (res *NewsList, err error) {
	ul := "https://api.weixin.qq.com/cgi-bin/material/batchget_material?access_token=" + token
	err = pkg.PostSchema(pkg.KindJson, ul, map[string]interface{}{
		"type":   NEWS,
		"offset": offset,
		"count":  limit,
	}, &res)
	return
}
