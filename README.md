

# シンプルなロガー

## 主な特徴
* contextを利用可能
    * 主にWEB APIでリクエストでセットした値をロガー側で利用する事を想定
* コードのファイル行数を出力
* fmtとslogの両方のモード
* Google loggingへ以下の情報をセット（デフォルトでオフ）
    * ラベル
    * ファイル
    * トレース情報
* 出力メッセージのカスタマイズ


## 例
```go
package main

import (
	"context"
	"strconv"

	simplelog "github.com/megur0/simple-logger"
)

func main() {
	l := simplelog.New(simplelog.LOG_MODE_FMT, simplelog.LOG_LEVEL_DEBUG, &simplelog.MyHandler{}, true)
	ctx := context.Background()
	ctx = context.WithValue(ctx, "requestId", "1234567890")
	l.Debug(ctx, "debug message")
}

type MyHandler struct{}

func (h *MyHandler) GetMessage(logger simplelog.Logger, c context.Context, level string, file string, line int, originalOutput string) string {
	// This is sample code customizing log message
	requestId := c.Value("requestId").(string)
	out := "[" + level + "]" + "[" + requestId + "]" + file + ":" + strconv.Itoa(line) + " " + originalOutput

	return out
}

func (h *MyHandler) GetLabels(logger simplelog.Logger, c context.Context) map[string]string {
	labels := map[string]string{}

	// set lebels if use gcp loging

	return labels
}

func (h *MyHandler) GetTrace(logger simplelog.Logger, c context.Context) string {
	// set lebels if use gcp loging

	return ""
}
```
