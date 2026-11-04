# orgboss 主要ロジック シーケンス図

## 1. 新規組織（Organization）作成

新規ユーザー登録時に、OrganizationとUserをトランザクションで同時作成する。

```mermaid
sequenceDiagram
    participant Client as クライアント
    participant App as アプリケーション
    participant Authboss as Authboss
    participant Manager as orgboss.Manager
    participant DB as データベース
    participant Hook as BeforeCreateHook

    Client->>App: 新規ユーザー登録リクエスト
    App->>Authboss: ユーザー登録開始
    Authboss->>Manager: CreateOrganizationWithUser()
    
    Manager->>Hook: BeforeOrganizationCreate()
    Hook-->>Manager: 続行
    
    Manager->>DB: トランザクション開始
    Manager->>DB: Organization作成（organization_idは主キー制約で自動的にユニーク）
    Manager->>DB: User作成（role=manager）
    Manager->>DB: トランザクションコミット
    Manager->>Hook: AfterOrganizationCreate()
    Manager-->>Authboss: 成功
    Authboss-->>App: ユーザー登録成功
    App-->>Client: 登録完了
```

## 2. 組織への招待（単一・バルク）

managerがuserを招待する。バルク招待にも対応。

```mermaid
sequenceDiagram
    participant Manager as manager
    participant App as アプリケーション
    participant ManagerAPI as orgboss.Manager
    participant DB as データベース
    participant EmailSender as EmailSender
    participant Hook as BeforeInviteHook

    Manager->>App: 招待リクエスト（メールアドレス）
    App->>ManagerAPI: InviteUser(email, orgID)
    
    ManagerAPI->>Hook: BeforeInvite()
    Hook-->>ManagerAPI: 続行
    
    ManagerAPI->>ManagerAPI: トークン生成（ハッシュ）
    ManagerAPI->>ManagerAPI: 有効期限計算（Config.InvitationExpiryDuration）
    ManagerAPI->>DB: Invitation作成（status=pending）
    
    ManagerAPI->>EmailSender: SendInvitation(email, token)
    EmailSender->>EmailSender: メール送信（SMTP）
    EmailSender-->>ManagerAPI: 送信完了
    
    ManagerAPI->>Hook: AfterInvite()
    ManagerAPI-->>App: 招待成功
    App-->>Manager: 招待完了

    Note over Manager,EmailSender: バルク招待の場合、複数のメールアドレスを<br/>ループ処理（MaxBulkInviteCountまで）
```

## 3. 招待の承諾・拒否

userが招待メールのリンクをクリックして承諾または拒否する。

```mermaid
sequenceDiagram
    participant User as user
    participant App as アプリケーション
    participant ManagerAPI as orgboss.Manager
    participant DB as データベース
    participant Authboss as Authboss

    User->>App: 招待リンククリック（トークン）
    App->>ManagerAPI: AcceptInvitation(token)
    
    Manager->>DB: Invitation検索（token, status=pending）
    alt トークンが見つからない or 期限切れ
        ManagerAPI-->>App: エラー（無効なトークン）
        App-->>User: エラーメッセージ
    else 有効なトークン
        ManagerAPI->>DB: トランザクション開始
        ManagerAPI->>DB: Invitation更新（status=accepted）
        ManagerAPI->>DB: User作成（organization_id, role=user）
        ManagerAPI->>DB: トランザクションコミット
        ManagerAPI-->>App: 承諾成功
        App->>Authboss: ユーザー登録完了通知
        App-->>User: 承諾完了（ログイン可能）
    end

    Note over User,Authboss: 拒否の場合も同様のフローで<br/>status=rejectedに更新
```

## 4. ユーザー退会

user自身の退会、managerによるuser削除、manager退会の3パターン。

```mermaid
sequenceDiagram
    participant Actor as アクター<br/>(user/manager)
    participant App as アプリケーション
    participant ManagerAPI as orgboss.Manager
    participant DeletionHandler as DeletionHandler
    participant DB as データベース
    participant Hook as BeforeDeleteHook

    Actor->>App: 退会リクエスト
    App->>ManagerAPI: DeleteUser(userID, orgID)
    
    ManagerAPI->>ManagerAPI: 権限チェック
    alt 権限なし
        ManagerAPI-->>App: エラー（権限なし）
        App-->>Actor: エラーレスポンス
    else 権限あり
        ManagerAPI->>Hook: BeforeUserDelete()
        Hook-->>ManagerAPI: 続行
        
        ManagerAPI->>ManagerAPI: ロール判定
        alt manager退会
            ManagerAPI->>DeletionHandler: DeleteOrganization(orgID)
            DeletionHandler->>DB: Organization削除（論理/物理）
            DeletionHandler->>DB: 関連User削除（論理/物理/マスク）
            DeletionHandler-->>ManagerAPI: 削除完了
        else user退会
            ManagerAPI->>DeletionHandler: DeleteUser(userID)
            DeletionHandler->>DB: User削除（論理/物理/マスク）
            DeletionHandler-->>ManagerAPI: 削除完了
        end
        
        ManagerAPI->>Hook: AfterUserDelete()
        ManagerAPI-->>App: 退会成功
        App-->>Actor: 退会完了
    end
```

## 5. 権限チェック（ミドルウェア）

HTTPリクエスト時にorganization_idとロールを検証する。

```mermaid
sequenceDiagram
    participant Client as クライアント
    participant Middleware as orgboss.Middleware
    participant Authboss as Authboss
    participant ManagerAPI as orgboss.Manager
    participant RoleChecker as RoleChecker
    participant Handler as リクエストハンドラー

    Client->>Middleware: HTTPリクエスト
    Middleware->>Authboss: 認証情報取得
    Authboss-->>Middleware: User情報（userID, organizationID, role）
    
    Middleware->>ManagerAPI: ValidateOrganizationAccess(userID, orgID)
    ManagerAPI->>DB: User検索（userID, organizationID）
    alt organization_id不一致
        ManagerAPI-->>Middleware: エラー（アクセス拒否）
        Middleware-->>Client: 403 Forbidden
    else organization_id一致
        ManagerAPI-->>Middleware: アクセス許可
        
        Middleware->>RoleChecker: CheckPermission(role, action)
        alt 権限不足
            RoleChecker-->>Middleware: エラー（権限なし）
            Middleware-->>Client: 403 Forbidden
        else 権限あり
            RoleChecker-->>Middleware: 許可
            Middleware->>Handler: リクエスト処理
            Handler-->>Middleware: レスポンス
            Middleware-->>Client: レスポンス
        end
    end
```

## 6. 招待の再発行

managerが既存の招待を再送信する。

```mermaid
sequenceDiagram
    participant Manager as manager
    participant App as アプリケーション
    participant ManagerAPI as orgboss.Manager
    participant DB as データベース
    participant EmailSender as EmailSender

    Manager->>App: 招待再発行リクエスト（invitationID）
    App->>ManagerAPI: ResendInvitation(invitationID, orgID)
    
    ManagerAPI->>ManagerAPI: 権限チェック（managerのみ）
    alt 権限なし
        ManagerAPI-->>App: エラー（権限なし）
        App-->>Manager: エラーレスポンス
    else 権限あり
        ManagerAPI->>DB: Invitation検索（invitationID, orgID）
        alt 招待が見つからない
            ManagerAPI-->>App: エラー（招待不存在）
            App-->>Manager: エラーレスポンス
        else 招待存在
            ManagerAPI->>ManagerAPI: トークン再生成（オプション）
            ManagerAPI->>ManagerAPI: 有効期限更新
            ManagerAPI->>DB: Invitation更新（status=pending, expiresAt更新）
            
            ManagerAPI->>EmailSender: SendInvitation(email, token)
            EmailSender->>EmailSender: メール送信（SMTP）
            EmailSender-->>ManagerAPI: 送信完了
            
            ManagerAPI-->>App: 再発行成功
            App-->>Manager: 再発行完了
        end
    end
```

## 7. プロフィール更新

manager、userが自分のプロフィールを更新する。

```mermaid
sequenceDiagram
    participant User as user/manager
    participant App as アプリケーション
    participant Middleware as orgboss.Middleware
    participant ManagerAPI as orgboss.Manager
    participant DB as データベース
    participant Authboss as Authboss

    User->>App: プロフィール更新リクエスト
    App->>Middleware: リクエスト処理
    Middleware->>Middleware: organization_id検証
    Middleware->>ManagerAPI: UpdateProfile(userID, orgID, data)
    
    ManagerAPI->>ManagerAPI: 権限チェック（自分のみ更新可能）
    alt 権限なし
        ManagerAPI-->>App: エラー（権限なし）
        App-->>User: エラーレスポンス
    else 権限あり
        ManagerAPI->>DB: User更新（organization_id一致確認）
        ManagerAPI->>Authboss: ユーザー情報更新通知
        ManagerAPI-->>App: 更新成功
        App-->>User: 更新完了
    end
```

## 8. Organization更新

managerがOrganizationの情報（組織名、ロゴなど）を更新する。

```mermaid
sequenceDiagram
    participant Manager as manager
    participant App as アプリケーション
    participant Middleware as orgboss.Middleware
    participant ManagerAPI as orgboss.Manager
    participant DB as データベース
    participant Hook as BeforeUpdateHook

    Manager->>App: Organization更新リクエスト（組織名、ロゴ等）
    App->>Middleware: リクエスト処理
    Middleware->>Middleware: organization_id検証
    Middleware->>ManagerAPI: UpdateOrganization(orgID, userID, data)
    
    ManagerAPI->>ManagerAPI: 権限チェック（managerのみ）
    alt 権限なし
        ManagerAPI-->>App: エラー（権限なし）
        App-->>Manager: エラーレスポンス
    else 権限あり
        ManagerAPI->>Hook: BeforeOrganizationUpdate()
        Hook-->>ManagerAPI: 続行
        
        ManagerAPI->>DB: Organization更新（organization_id一致確認、組織名の重複は許可）
        
        ManagerAPI->>Hook: AfterOrganizationUpdate()
        ManagerAPI-->>App: 更新成功
        App-->>Manager: 更新完了
    end
```

