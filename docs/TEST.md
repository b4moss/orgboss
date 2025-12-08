# orgboss テスト仕様書

## CreateOrganizationWithUser

- BeforeOrganizationCreateフック実行
- トランザクション開始
- Organization作成（organization_idは主キー制約で自動的にユニーク）
- User作成（role=manager）
- トランザクションコミット
- AfterOrganizationCreateフック実行

### テストシナリオ

**正常系**
- OrganizationとUserが正常に作成され、Userのroleがmanagerになる
- BeforeOrganizationCreateとAfterOrganizationCreateフックが実行される
- トランザクションが正常にコミットされる

**異常系**
- Organization作成失敗時にトランザクションがロールバックされ、Userも作成されない
- BeforeOrganizationCreateフックでエラーが発生した場合、処理が中断される

## InviteUser

- BeforeInviteフック実行
- トークン生成（ハッシュ）
- 有効期限計算（Config.InvitationExpiryDuration）
- Invitation作成（status=pending）
- メール送信（EmailSender.SendInvitation）
- AfterInviteフック実行

### テストシナリオ

**正常系**
- Invitationが正常に作成され、statusがpendingになる
- トークンが生成され、有効期限が正しく設定される
- メールが正常に送信される

**異常系**
- メール送信失敗時にエラーが返される
- BeforeInviteフックでエラーが発生した場合、処理が中断される

## InviteUsers（バルク招待）

- 招待数上限チェック（MaxBulkInviteCount）
- 各メールアドレスに対してInviteUserを実行

### テストシナリオ

**正常系**
- 上限内のメールアドレスが正常に招待される
- 各メールアドレスに対してInvitationが作成される

**異常系**
- 招待数がMaxBulkInviteCountを超える場合、エラーが返される
- 一部のメールアドレスでエラーが発生しても、他の招待は処理される

## AcceptInvitation

- トークン検証（Invitation検索）
- 有効期限チェック
- トランザクション開始
- Invitation更新（status=accepted）
- User作成（organization_id, role=user）
- トランザクションコミット

### テストシナリオ

**正常系**
- 有効なトークンで正常に承諾され、Userが作成される
- Invitationのstatusがacceptedに更新される
- トランザクションが正常にコミットされる

**異常系**
- 無効なトークンの場合、エラーが返される
- 有効期限切れの場合、エラーが返される
- 既にacceptedまたはrejectedの場合、エラーが返される

## RejectInvitation

- トークン検証（Invitation検索）
- 有効期限チェック
- Invitation更新（status=rejected）

### テストシナリオ

**正常系**
- 有効なトークンで正常に拒否され、Invitationのstatusがrejectedに更新される

**異常系**
- 無効なトークンの場合、エラーが返される
- 有効期限切れの場合、エラーが返される

## DeleteUser

- 権限チェック（ValidateOrganizationAccess）
- BeforeUserDeleteフック実行
- ロール判定
- manager退会の場合：DeleteOrganization実行
- user退会の場合：DeleteUser実行（DeletionHandler）
- AfterUserDeleteフック実行

### テストシナリオ

**正常系**
- userが自ら退会する場合、正常に削除される
- managerがuserを退会させる場合、正常に削除される
- managerが退会する場合、Organizationも削除される

**異常系**
- 権限がない場合、エラーが返される
- organization_idが一致しない場合、エラーが返される

## DeleteOrganization

- Organization削除（論理/物理、DeletionHandler）
- 関連User削除（論理/物理/マスク、DeletionHandler）

### テストシナリオ

**正常系**
- Organizationと関連Userが正常に削除される（DeletionHandlerの実装に従う）

**異常系**
- 削除処理でエラーが発生した場合、エラーが返される

## ResendInvitation

- 権限チェック（managerのみ）
- Invitation検索（invitationID, orgID）
- トークン再生成（オプション）
- 有効期限更新
- Invitation更新（status=pending, expiresAt更新）
- メール再送信（EmailSender.SendInvitation）

### テストシナリオ

**正常系**
- managerが正常に招待を再送信できる
- Invitationのstatusがpendingに更新され、有効期限が更新される
- メールが正常に再送信される

**異常系**
- userが実行した場合、エラーが返される
- 招待が存在しない場合、エラーが返される
- メール送信失敗時にエラーが返される

## UpdateProfile

- 権限チェック（自分のみ更新可能）
- organization_id一致確認
- User更新
- Authboss通知

### テストシナリオ

**正常系**
- 自分のプロフィールが正常に更新される
- Authbossに通知が送られる

**異常系**
- 他人のプロフィールを更新しようとした場合、エラーが返される
- organization_idが一致しない場合、エラーが返される

## UpdateOrganization

- 権限チェック（managerのみ）
- BeforeOrganizationUpdateフック実行
- organization_id一致確認
- Organization更新（組織名の重複は許可）
- AfterOrganizationUpdateフック実行

### テストシナリオ

**正常系**
- managerが正常にOrganizationを更新できる
- 組織名の重複は許可される
- BeforeOrganizationUpdateとAfterOrganizationUpdateフックが実行される

**異常系**
- userが実行した場合、エラーが返される
- organization_idが一致しない場合、エラーが返される

## ValidateOrganizationAccess

- User検索（userID, organizationID）
- organization_id一致確認

### テストシナリオ

**正常系**
- organization_idが一致する場合、アクセスが許可される

**異常系**
- organization_idが一致しない場合、エラーが返される
- ユーザーが存在しない場合、エラーが返される

## CheckPermission

- ロール判定（RoleChecker）
- アクション権限チェック

### テストシナリオ

**正常系**
- managerがOrganizationのUpdate/Delete権限を持つ
- userがOrganizationのRead権限を持つ

**異常系**
- userがUpdate/Deleteを実行しようとした場合、エラーが返される

## 共通処理：トークン生成

- ハッシュ生成（セキュアなランダム文字列）

### テストシナリオ

**正常系**
- セキュアなランダム文字列が生成される
- 毎回異なるトークンが生成される

## 共通処理：有効期限計算

- 現在時刻 + Config.InvitationExpiryDuration

### テストシナリオ

**正常系**
- Config.InvitationExpiryDurationに基づいて正しく有効期限が計算される

## 共通処理：有効期限チェック

- 現在時刻とExpiresAtの比較

### テストシナリオ

**正常系**
- 有効期限内の場合、trueが返される
- 有効期限切れの場合、falseが返される