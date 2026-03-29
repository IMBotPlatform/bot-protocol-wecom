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
- `(*LongConnBot).Start`
- `(*LongConnBot).Close`
- `(*LongConnBot).SendMarkdown`
- `(*LongConnBot).SendTemplateCard`
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
- `BuildLongConnSendTemplateCardRequest`
