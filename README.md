# FTL
## これはなに？

誰でも自由にキャッシュサーバを設置できて、誰でも自由にコンテンツ配信に利用できて、利用者はその存在を全く意識せずに最適なサーバーからダウンロードできる、そんなオープンな CDN 基盤できないかなという思いつきで実装してみた CDN 基盤ソフトウェアです。

## 使い方

### キャッシュサーバを設置する場合

```
$ sudo iptables -t nat -F
$ sudo iptables -t nat -A PREROUTING -p udp --dport 53 \
-j REDIRECT --to-port 5353
$ sudo iptables -t nat -A PREROUTING -p tcp --dport 80 \
-j REDIRECT --to-port 8080
$ go get github.com/m-asama/ftl/ftld
$ ${GOPATH}/bin/ftld -hints 192.0.2.12 \
           -myNodeId 198.51.100.23 \
           -myDataDir data -logDir log
```

FTL は DNS 権威サーバ機能と HTTP プロキシサーバ機能を提供します。ただし、一般ユーザ権限で実行できるように、それぞれ UDP/5353 と TCP/8080 でサービスを提供します。最初の 3 つの `iptables` で、 UDP/5353 を UDP/53 に、 TCP/8080 を TCP/80 に、それぞれポートリダイレクトする設定をします。すでに他のポートリダイレクトの設定がされているときは `iptables -t nat -F` でその設定が消されてしまうので注意してください。

FTL は Go 言語で書かれています。 Go の環境を用意し `go get` でサービスプログラムである `ftld` を取得し、実行します。 `-hints` はすでに稼働している FTL ノードの IPv4 アドレスを指定します。 `ftld` は起動後にこのヒントノードに接続し FTL の CDN を構成する全ての FTL ノードの一覧を取得しそれらの FTL ノードと通信します。 `-myNodeId` には自身のグローバル IPv4 アドレスを指定します。ここで指定した IPv4 アドレスに対して他の FTL ノードは接続を試みるので必ずグローバル IPv4 アドレスでなければなりません。 FTL ノードがプライベート IPv4 アドレスしか持たないときはここで指定するグローバル IPv4 アドレスへの UDP/53、 TCP/80、 TCP/9000(これは FTL ノード間での通信に用いられます) が FTL ノードへ転送される必要があります。 `-myDataDir` にはキャッシュとして保持するデータを保管するディレクトリを指定します。 `-logDir` にはエラーログとアクセスログを保存するディレクトリを指定します。

### コンテンツ配信に使う場合

例えば `example.jp` という DNS ドメイン名で FTL を利用したい場合は `example.jp` の DNS ゾーンに以下のような NS レコードを登録します。対応する A レコードには稼働している FTL ノードの IPv4 アドレスを指定します。

```
;; AUTHORITY SECTION:
ftl-cache.example.jp.  60  IN  NS  ftl1.example.jp.
ftl-cache.example.jp.  60  IN  NS  ftl2.example.jp.
ftl-cache.example.jp.  60  IN  NS  ftl3.example.jp.

;; ADDITIONAL SECTION:
ftl1.example.jp.       60  IN  A   192.0.2.12
ftl2.example.jp.       60  IN  A   198.51.100.23
ftl3.example.jp.       60  IN  A   203.0.113.34
```

そして FTL を使って配信したいコンテンツの URL のホスト名に `ftl-cache` を付け加えます。例えば `http://www.example.jp/files.zip` というファイルを FTL で配信したいときは `http://www.ftl-cache.example.jp/files.zip` という URL で配信します。 `ftl-cache.example.jp` という DNS ゾーンの権威サーバには FTL ノードが設定されているので `www.ftl-cache.example.jp` の DNS 名前解決は FTL ノードに問い合わせが行きます。 FTL ノードは問い合わせをしてきた DNS キャッシュサーバの接続元 IPv4 アドレスから最も近いと思われる FTL ノードの IPv4 アドレスを回答として返します(最も近いと思われる FTL ノードの求め方は後述)。

その DNS 応答を受け取ったクライアントは回答として返された FTL ノードに対して HTTP リクエストを投げます。 HTTP リクエストを受け取った FTL ノードはそのリクエストのキャッシュを保持している場合はそのキャッシュをクライアントへ返します。キャッシュが存在しなかったり存在しても古かったりしたときは HTTP ヘッダのホスト情報から ftl-cache を除いたホストへ HTTP リクエストをプロキシし、結果をクライアントへ返します。

## 最適な FTL ノードの推定

FTL では `ftl-cache` の他に `ftl-observation` という特別な DNS ゾーンを利用します。前述の通り `ftl-cache` はその DNS ゾーン内のホスト名への問い合わせにその問い合わせを行ってきた接続元 DNS キャッシュサーバに近い FTL ノードを回答します。 `ftl-observation` は `ftl-cache` と異なり今まで回答として返していない FTL ノードを回答します。その際、回答として返す FTL ノードに対して事前に「あなたを `ftl-observation` の回答として返しますよ」という通知を行います。通知を受け取った FTL ノードはその直後にきた `ftl-observation` を含むホスト名の HTTP リクエストを DNS キャッシュサーバと紐付け、コンテンツを返すのにかかった時間と転送容量から帯域幅を計算し記録します。`ftl-cache` のときはこの記録しておいた FTL ノードと DNS キャッシュサーバの間の帯域幅の情報を元に最も近いと思われる FTL ノードを推定します。

ということで FTL が最適な FTL ノードを推定するためには `ftl-observation` による帯域幅情報の収集が十分な数だけ行われている必要があります。そのためにも FTL を用いてコンテンツ配信を行う方や FTL のキャッシュサーバを設置する方は適度なサイズのユーザーエクスペリエンスに影響を与えないようなコンテンツを `ftl-observation` で配信するようにしてください。例えば真っ白な画像ファイルを `http://www.example.jp/blank.png` に置き、 `http://www.ftl-observation.example.jp/blank.png` を頻繁に読み込まれるサイトのヘッダやフッタに埋め込むことで帯域幅の情報を収集することができるようになります。

## ログの収集と統計情報の確認

`ftlctl` というコマンドでログの収集と統計情報を確認することができます。

ログの収集と統計情報の確認は FTL ノードの IP アドレスを一つでも知っていれば誰でも行うことができます。

```
$ go get github.com/m-asama/ftl/ftlctl
```

アクセスログを表示するには以下のように `ftlctl` を実行します。

```
$ ${GOPATH}/bin/ftlctl 192.0.2.12 log
...
2017-08-14 03:58:02.909906999 +0000 UTC 192.0.2.12 203.0.113.34:55362 GET 200 true "www.ftl-cache.example.jp" "/blank.png" 28760 67413
```

最初の引数は稼働している FTL ノードのどれか一つを指定します。 `ftlctl` は指定された FTL ノードに接続し FTL ノードの一覧を取得し、全ての FTL ノードから最新のアクセスログを収集し、表示します。

過去の日付のアクセスログを取得したいときは以下のように日付を指定します。

```
$ ${GOPATH}/bin/ftlctl 192.0.2.12 log 20170813
```

表示されるログの意味は以下の通りです。

|  | 例                                      | 内容                                 |
|-:|:----------------------------------------|:-------------------------------------|
| 1| 2017-08-14 03:58:02.909906999 +0000 UTC | アクセスがあった日時(UTC)            |
| 2| 192.0.2.12                              | HTTP リクエストを処理した FTL ノード |
| 3| 203.0.113.34:55362                      | 接続元 IP アドレスとポート番号       |
| 4| GET                                     | HTTP メソッド                        |
| 5| 200                                     | HTTP ステータスコード                |
| 6| true                                    | キャッシュにヒットしたか否か         |
| 7| "www.ftl-cache.example.jp"              | HTTP リクエストのホスト名            |
| 8| "/blank.png"                            | HTTP リクエストの URI                |
| 9| 28760                                   | 転送容量                             |
|10| 67413                                   | 転送時間(ナノ秒)                     |

統計情報を表示するには以下のように `ftlctl` を実行します。

```
$ ${GOPATH}/bin/ftlctl 192.0.2.12 stats
192.0.2.12       0.203.0.113         0.271910482    0.271910482    0.271910482
198.51.100.23    0.203.0.113         0.261169181    0.261169181    0.261169181
```

表示される統計情報の意味は以下の通りです。

|  | 例                | 内容                                                         |
|-:|:------------------|:-------------------------------------------------------------|
| 1| 192.0.2.12        | HTTP リクエストを処理した FTL ノード                         |
| 2| 0.203.0.113       | DNS キャッシュサーバの IPv4 アドレスを 8bit シフトさせたもの |
| 3| 0.271910482       | 1 時間平均の転送速度                                         |
| 4| 0.271910482       | 1 日平均の転送速度                                           |
| 5| 0.271910482       | 1 週間平均の転送速度                                         |

## TODO

* DNS 権威サーバ機能は適当に実装しているのでちゃんとする
* HTTP 転送帯域幅情報の正確な収集方法の検討
* DNS キャッシュサーバと HTTP アクセスの紐付けアルゴリズムの再検討
* 最適な FTL ノードの推定アルゴリズムの再検討
