import type { AdminUser } from '../../types'
import { displayUserLabel, formatCredits, formatDateTime, readNumberInput, type NumericInputValue } from './adminHelpers'

type UsersTabProps = {
  users: AdminUser[]
  filteredUsers: AdminUser[]
  usersLoading: boolean
  userQuery: string
  selectedLedgerUser: string
  ledgerLoading: boolean
  grantUsername: string
  grantAmount: NumericInputValue
  grantReason: string
  grantSubmitting: boolean
  roleBusyUser: string
  onUserQueryChange: (value: string) => void
  onRefreshUsers: () => void
  onGrantUsernameChange: (value: string) => void
  onGrantAmountChange: (value: NumericInputValue) => void
  onGrantReasonChange: (value: string) => void
  onSubmitGrantCredits: () => void
  onLoadLedger: (username: string) => void
  onToggleAdminRole: (user: AdminUser) => void
}

export function UsersTab({
  users,
  filteredUsers,
  usersLoading,
  userQuery,
  selectedLedgerUser,
  ledgerLoading,
  grantUsername,
  grantAmount,
  grantReason,
  grantSubmitting,
  roleBusyUser,
  onUserQueryChange,
  onRefreshUsers,
  onGrantUsernameChange,
  onGrantAmountChange,
  onGrantReasonChange,
  onSubmitGrantCredits,
  onLoadLedger,
  onToggleAdminRole,
}: UsersTabProps) {
  return (
    <section className="admin-tab-panel admin-users-section admin-users-panel" id="admin-panel-users" role="tabpanel" aria-labelledby="admin-tab-users">
      <div className="admin-section-heading admin-users-heading">
        <div>
          <h2 id="admin-users-title">用户管理</h2>
          <p className="muted">查看余额、加次数流水和管理员角色。当前展示 {filteredUsers.length} / {users.length} 个用户。</p>
        </div>
        <div className="admin-users-tools">
          <label className="admin-user-search">
            <span>搜索用户</span>
            <input value={userQuery} onChange={(event) => onUserQueryChange(event.target.value)} placeholder="用户名、显示名或邮箱" />
          </label>
          <button type="button" onClick={onRefreshUsers} disabled={usersLoading}>{usersLoading ? '刷新中...' : '刷新用户'}</button>
        </div>
      </div>
      <div
        className="admin-grant-form"
        onKeyDown={(event) => {
          if (event.key === 'Enter') {
            event.preventDefault()
            onSubmitGrantCredits()
          }
        }}
      >
        <label>
          用户
          <select value={grantUsername} onChange={(event) => onGrantUsernameChange(event.target.value)} disabled={grantSubmitting || usersLoading}>
            <option value="">选择用户</option>
            {users.map((user) => (
              <option key={user.username} value={user.username}>{displayUserLabel(user)}</option>
            ))}
          </select>
        </label>
        <label>
          增加次数
          <input type="number" min={1} step={1} value={grantAmount} onChange={(event) => onGrantAmountChange(readNumberInput(event.target.value))} />
        </label>
        <label>
          原因 <span className="required-mark">必填</span>
          <input value={grantReason} onChange={(event) => onGrantReasonChange(event.target.value)} placeholder="例如：线下付款补录" />
        </label>
        <button type="button" className="primary" onClick={onSubmitGrantCredits} disabled={grantSubmitting || usersLoading}>{grantSubmitting ? '提交中...' : '增加次数'}</button>
      </div>
      <div className="profile-table-wrap admin-users-table-wrap">
        <table className="admin-users-table">
          <thead>
            <tr>
              <th>用户</th>
              <th>邮箱</th>
              <th>余额</th>
              <th>管理员</th>
              <th>注册时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {usersLoading && users.length === 0 ? (
              <tr><td colSpan={6}>正在读取用户...</td></tr>
            ) : filteredUsers.length === 0 ? (
              <tr><td colSpan={6}>{users.length ? '没有匹配用户' : '暂无用户数据'}</td></tr>
            ) : filteredUsers.map((user) => (
              <tr key={user.username}>
                <td>
                  <strong>{user.displayName || user.username}</strong>
                  <span>{user.username}</span>
                </td>
                <td>{user.email || '-'}</td>
                <td className="numeric-cell">{formatCredits(user.creditsBalance)}</td>
                <td><span className={user.isAdmin ? 'admin-role-badge admin-role-badge-on' : 'admin-role-badge'}>{user.isAdmin ? '管理员' : '普通用户'}</span></td>
                <td>{formatDateTime(user.createdAt)}</td>
                <td>
                  <div className="admin-row-actions">
                    <button type="button" onClick={() => onLoadLedger(user.username)} disabled={ledgerLoading && selectedLedgerUser === user.username}>流水</button>
                    <button type="button" onClick={() => onToggleAdminRole(user)} disabled={roleBusyUser === user.username}>{roleBusyUser === user.username ? '处理中...' : user.isAdmin ? '取消管理员' : '设为管理员'}</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  )
}
