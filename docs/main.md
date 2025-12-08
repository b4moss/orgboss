# orgboss

Go言語のAuthbossパッケージと連動する、組織管理・マルチテナント管理パッケージ

organization_idとuser_idを紐付け、テナントとする。

## 簡易仕様

1. 新規にユーザーを作成する時には、新規にOrganizationの作成も行う(トランザクション)
2. organization_idはデータベースの主キー制約によりユニークである（組織名の重複は許可される）
3. 最初にOrganizationと同時に作られたユーザーは、強制的にmanagerロール
4. managerは、organizationのUpdate、Delete権限を持つ
5. managerの他には、userロール。userロールは、Organizationの情報は、Readしかできない

## ユースケース

### アクター

1. manager
2. user
3. Administrator（orgbossの上位の概念で登場するユーザーロール）

### 新規組織（Organization）作成

1. manager（この時点では、まだmanagerではないが、便宜上）は、Organizationを作成できる
  - 実際のアプリの振る舞いとしては、単なる新規会員登録に見せる。裏でトランザクションで、Organizationも作られるだけ
2. managerは、正しくOrganizationが作成された後、ログインができる
  - 逆に言えば、何らかの理由でOrganizationが作成されなかった場合は、ユーザーとしてもログインできない

### 組織への招待

1. managerは、userを招待できる。招待はメールで送られ、一定時間内に記載されたリンクのクリックがなければ、無効化される。
  - したがって、招待にはuserのメールアドレスを知っている必要がある。
  - この機能を利用するアプリは、SMTPサーバーと接続されていなければならない。
  - 数の上限はあるとして、バルク招待ができると良い。
2. managerは、userへの招待を再発行（再送信）できる
  - 任意のタイミングで、再発行できる
3. 招待を受け取ったuserは、承諾するにはリンクをクリック。拒否するには、拒否リンクをクリックする
  - よくある、URLにハッシュを含む、あれ

### ログイン・ログアウト

以下は、Authbossの標準機能で実装される

1. manager、userはログインができる
2. manager、userはログアウトができる
3. 認証は、パスワードとメールアドレスを用いる

### パスワードリセット

1. manager、userはパスワードリセット機能を用いることができる
  - パスワードリセット後は、登録メールアドレスに記載されたURLをクリックすることでパスワード変更画面に遷移
  - そこで再設定を行い、ログインできる。

### ユーザー管理

1. manager、userは、自分のプロフィール設定を更新できる
2. managerは、Organizationの情報を更新できる（組織名、ロゴなど）
3. userは、自ら退会することができる
4. managerは、userを退会させることができる
5. managerが退会する時は、Organizationも削除される

### 閲覧権限

1. organization_idが違うユーザー同士は、他組織の全てのモデルのCRUDおよびその他の処理を実行することはできない
2. 設定を変えると、Readやコメントの追加などができる（マッチングサイトやポータルサイトを設定したイメージ）

## 設計方針

### アーキテクチャ

- **ライブラリ型**：コア機能は固定、カスタマイズ可能な部分は設定構造体とインターフェースで提供
- **GORM使用**：データベース操作はGORMで実装
- **設定管理**：環境変数（.env）と設定構造体の組み合わせ

### カスタマイズ戦略

アプリごとの細かな挙動変更に対応するため、以下の3つの方法でカスタマイズを可能にする：

1. **設定構造体**：招待有効期限、デフォルトロールなど、よく変更される項目
2. **インターフェース**：退会処理、ロール判定、重複チェックなど、挙動を変えたい処理
3. **フックポイント**：Organization作成前後、招待送信前後など、追加処理を挿入可能

## データモデル設計

### Organization

- `ID`：主キー（ユニーク制約）
- `Name`：組織名（重複可、同名の企業が存在することを想定）
- `CreatedAt`：作成日時
- `UpdatedAt`：更新日時
- `DeletedAt`：削除日時（論理削除対応）

### User（AuthbossのUserモデルを拡張）

- Authbossの標準Userフィールド
- `OrganizationID`：外部キー（Organizationへの参照）
- `Role`：ロール（manager/user、拡張可能）

### Invitation

- `ID`：主キー
- `Email`：招待先メールアドレス
- `OrganizationID`：外部キー（Organizationへの参照）
- `Token`：招待トークン（ハッシュ）
- `ExpiresAt`：有効期限（設定可能）
- `Status`：ステータス（pending/accepted/rejected/expired）
- `CreatedAt`：作成日時

### OrganizationUser（中間テーブル、必要に応じて）

- `OrganizationID`：外部キー
- `UserID`：外部キー
- `Role`：ロール
- `JoinedAt`：参加日時

## カスタマイズ可能な設定

### Config構造体

以下の設定項目を提供し、環境変数や設定ファイルから読み込めるようにする：

- `InvitationExpiryDuration`：招待の有効期限（デフォルト24時間）
- `DefaultRole`：新規ユーザーのデフォルトロール
- `EnableBulkInvite`：バルク招待の有効/無効
- `MaxBulkInviteCount`：バルク招待の最大数

### インターフェース

カスタマイズ可能な処理をインターフェースで抽象化：

- `RoleChecker`：ロール判定ロジック（デフォルト：manager/user、拡張可能）
- `DeletionHandler`：退会処理（物理削除/論理削除/メールマスク等）
- `EmailSender`：メール送信（SMTP連携）

### フックポイント

以下のタイミングで追加処理を実行可能：

- Organization作成前後
- ユーザー作成前後
- 招待送信前後
- 退会処理前後

## パッケージ構造

```
orgboss/
├── config.go          # 設定構造体とデフォルト値
├── models.go          # Organization, User, Invitation, Role等のモデル
├── manager.go         # コア機能の実装（組織作成、招待、権限チェック等）
├── interfaces.go      # カスタマイズ用インターフェース
├── hooks.go           # フックポイント定義
├── errors.go          # エラー定義
└── middleware.go      # HTTPミドルウェア（organization_idの検証等）
```

## コア機能実装

### 1. 組織・ユーザー作成

- トランザクションでOrganizationとUserを同時作成
- 最初のユーザーは自動的にmanagerロール
- organization_idはデータベースの主キー制約により自動的にユニーク（組織名の重複は許可される）
- Organization作成失敗時はユーザーも作成されない（ログイン不可）

### 2. 招待機能

- トークン生成（ハッシュ）
- 有効期限管理（`Config.InvitationExpiryDuration`で設定可能）
- メール送信（`EmailSender`インターフェース実装）
- バルク招待対応（`Config.EnableBulkInvite`、`Config.MaxBulkInviteCount`で制御）
- 招待の再発行（再送信）機能

### 3. 権限管理

- organization_idベースのアクセス制御
- ロールベースの権限チェック（`RoleChecker`インターフェースで拡張可能）
- HTTPミドルウェアで自動検証
- manager：OrganizationのUpdate、Delete権限
- user：OrganizationのRead権限のみ

### 4. ユーザー管理

- プロフィール更新（manager、user共通）
- Organization更新（managerのみ、組織名、ロゴなど）
- 退会処理（`DeletionHandler`インターフェースでカスタマイズ可能）
  - user：自ら退会可能
  - manager：userを退会させることができる
  - manager退会時：Organizationも削除（`DeletionHandler`でカスタマイズ可能）

## Authboss連携

- AuthbossのUserモデルを拡張して`organization_id`と`role`を追加
- Authbossのフックポイントを利用して組織作成を連動
- 認証後のミドルウェアで`organization_id`を検証
- ログイン・ログアウト、パスワードリセットはAuthbossの標準機能を使用

## 実装の優先順位

1. **Phase 1**: データモデルと基本CRUD
2. **Phase 2**: 組織作成とユーザー作成の連動
3. **Phase 3**: 招待機能
4. **Phase 4**: 権限管理とミドルウェア
5. **Phase 5**: カスタマイズ用インターフェース実装
6. **Phase 6**: Authboss連携
7. **Phase 7**: テストとドキュメント
