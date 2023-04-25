package handler

import (
	"log"
	"strings"
	"time"

	"github.com/walkerdu/weixin-backend/internal/pkg/chatbot"
	"github.com/walkerdu/weixin-backend/pkg/wechat"
)

const WeChatTimeOutSecs = 5

func init() {
	handler := &TextMessageHandler{
		// 用户消息处理结果的cache，超过5s，就cache住, 等待用户指令进行推送
		rspMsgCache: make(map[string]struct {
			content string // cache的回复内容
			msgId   int64  // cache此回复对应的request的message ID
		}),
	}

	HandlerInst().RegisterLogicHandler(wechat.MessageTypeText, handler)
}

type TextMessageHandler struct {
	rspMsgCache map[string]struct {
		content string
		msgId   int64
	} // 用户消息处理结果的cache，超过5s，就cache住, 等待用户指令进行推送
}

func (t *TextMessageHandler) GetHandlerType() wechat.MessageType {
	return wechat.MessageTypeText
}

func (t *TextMessageHandler) HandleMessage(msg wechat.MessageIF) (wechat.MessageIF, error) {
	textMsg := msg.(*wechat.TextMessageReq)

	// 用户指令，直接从cache中读取
	if strings.TrimSpace(textMsg.Content) == "继续" {
		if cacheMsg, exist := t.rspMsgCache[textMsg.FromUserName]; !exist {
			return &wechat.TextMessageRsp{Content: "nothing to continue"}, nil
		} else {
			delete(t.rspMsgCache, textMsg.FromUserName)
			log.Printf("[INFO][HandleMessage] cache Response send to user, MsgId=%d", cacheMsg.msgId)
			return &wechat.TextMessageRsp{Content: cacheMsg.content}, nil
		}
	}

	begin := time.Now().Unix()
	chatRsp, err := chatbot.MustChatbot().GetResponse(textMsg.FromUserName, textMsg.Content)
	if err != nil {
		log.Printf("[ERROR][HandleMessage] chatbot.GetResponse failed, err=%s", err)
		chatRsp = "chatbot something wrong, please contact owner"
	}

	if time.Now().Unix()-begin >= WeChatTimeOutSecs {
		log.Printf("[WARN][HandleMessage] Response cost time too long, cache it, MsgId=%d", textMsg.MsgId)
		t.rspMsgCache[textMsg.FromUserName] = struct {
			content string
			msgId   int64
		}{
			content: chatRsp,
			msgId:   textMsg.MsgId,
		}
	}

	textMsgRsp := wechat.TextMessageRsp{
		Content: chatRsp,
	}

	return &textMsgRsp, nil
}
