package orgboss

import "orgboss/types"

// インターフェースも types パッケージのものを再エクスポート
type Storage = types.Storage
type RoleChecker = types.RoleChecker
type DeletionHandler = types.DeletionHandler
type EmailSender = types.EmailSender
