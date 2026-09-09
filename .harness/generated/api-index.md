# Generated API Index

## Package Inventory

- `github.com/IMBotPlatform/bot-protocol-wecom/pkg/wecom`
- `github.com/IMBotPlatform/bot-protocol-wecom/example/echo`

## Notable Exported Variables

- `ErrNoResponse`
- `NoResponse`

## Notable Exported Constructors

- `NewBot`
- `NewBotWithOptions`
- `NewLongConnBot`
- `NewLongConnBotWithOptions`
- `NewCrypt`

## Notable Exported Types

- `Bot`
- `StartOptions`
- `LongConnBot`
- `LongConnOptions`
- `LongConnHeaders`
- `LongConnRawFrame`
- `LongConnPushMessage`
- `LongConnChatType`
- `LongConnMediaType`
- `LongConnUploadMediaInitBody`
- `LongConnUploadMediaChunkBody`
- `LongConnUploadMediaFinishBody`
- `LongConnUploadMediaResult`
- `Crypt`
- `Context`
- `Chunk`
- `Handler`
- `Message`
- `EncryptedRequest`
- `EncryptedResponse`
- `TemplateCard`
- `LongConnRequest`
- `LongConnResponse`
- `MediaMessagePayload`
- `VideoMessagePayload`
- `FileMessage`
- `ImageMessage`
- `VoiceMessage`
- `VideoMessage`

## Notable Exported Methods

- `(*Bot).Start`
- `(*Bot).ServeHTTP`
- `(*Bot).Response`
- `(*Bot).ResponseMarkdown`
- `(*Bot).ResponseTemplateCard`
- `(*Bot).DecryptDownloadedFile`
- `(*Crypt).VerifyURL`
- `(*Crypt).DecryptMessage`
- `(*Crypt).EncryptResponse`
- `(*Crypt).Encrypt`
- `(*Crypt).Decrypt`
- `(*Crypt).DecryptDownloadedFile`
- `(*LongConnBot).Ready` / `ErrLongConnReplaced` / `LongConnOptions.OnError`
- `(*LongConnBot).Start`
- `(*LongConnBot).Close`
- `(*LongConnBot).SendMarkdown`
- `(*LongConnBot).SendMarkdownWithChatType`
- `(*LongConnBot).SendTemplateCard`
- `(*LongConnBot).SendTemplateCardWithChatType`
- `(*LongConnBot).SendFile`
- `(*LongConnBot).SendFileWithChatType`
- `(*LongConnBot).SendImage`
- `(*LongConnBot).SendImageWithChatType`
- `(*LongConnBot).SendVoice`
- `(*LongConnBot).SendVoiceWithChatType`
- `(*LongConnBot).SendVideo`
- `(*LongConnBot).SendVideoWithChatType`
- `(*LongConnBot).InitMediaUpload`
- `(*LongConnBot).UploadMediaChunk`
- `(*LongConnBot).FinishMediaUpload`
- `(*LongConnBot).UploadMedia`
- `(HandlerFunc).Handle`
- `(LongConnRawFrame).HasAckResult`
- `(LongConnRawFrame).IsCallback`
- `(LongConnRawFrame).UnmarshalBody`

## Notable Helper Functions

- `CalcSignature`
- `DecryptDownloadedFileWithAESKey`
- `BuildStreamReply`
- `BuildStreamReplyWithMsgItems`
- `BuildStreamImageItemFromBytes`
- `NewLongConnRequest`
- `BuildLongConnSubscribeRequest`
- `BuildLongConnPingRequest`
- `BuildLongConnSendMarkdownRequest`
- `BuildLongConnSendMarkdownRequestWithChatType`
- `BuildLongConnSendTemplateCardRequest`
- `BuildLongConnSendTemplateCardRequestWithChatType`
- `BuildLongConnSendFileRequest` / `BuildLongConnSendFileRequestWithChatType`
- `BuildLongConnSendImageRequest` / `BuildLongConnSendImageRequestWithChatType`
- `BuildLongConnSendVoiceRequest` / `BuildLongConnSendVoiceRequestWithChatType`
- `BuildLongConnSendVideoRequest` / `BuildLongConnSendVideoRequestWithChatType`
- `BuildLongConnUploadMediaInitRequest`
- `BuildLongConnUploadMediaChunkRequest`
- `BuildLongConnUploadMediaFinishRequest`
