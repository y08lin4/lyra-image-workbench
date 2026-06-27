import type { AdminUser, CreditLedgerEntry } from '../../types'
import { displayUserLabel, formatDateTime, formatDelta } from './adminHelpers'

type LedgerTabProps = {
  users: AdminUser[]
  selectedLedgerUser: string
  ledger: CreditLedgerEntry[]
  ledgerLoading: boolean
  usersLoading: boolean
  onSelectLedgerUser: (username: string) => void
  onClearLedgerUser: () => void
  onLoadLedger: (username: string) => void
}

export function LedgerTab({
  users,
  selectedLedgerUser,
  ledger,
  ledgerLoading,
  usersLoading,
  onSelectLedgerUser,
  onClearLedgerUser,
  onLoadLedger,
}: LedgerTabProps) {
  return (
    <section className="admin-tab-panel admin-ledger-section admin-ledger-panel" id="admin-panel-ledger" role="tabpanel" aria-labelledby="admin-tab-ledger">
      <div className="admin-section-heading compact admin-ledger-heading">
        <div>
          <h2>用户流水</h2>
          <p className="muted">{selectedLedgerUser ? `${selectedLedgerUser} 的额度变动记录` : '选择用户后查看额度变动记录。'}</p>
        </div>
        <div className="admin-users-tools admin-ledger-tools">
          <label className="admin-user-search">
            <span>用户</span>
            <select
              value={selectedLedgerUser}
              onChange={(event) => {
                const username = event.target.value
                if (username) {
                  onSelectLedgerUser(username)
                } else {
                  onClearLedgerUser()
                }
              }}
              disabled={usersLoading}
            >
              <option value="">选择用户</option>
              {users.map((user) => (
                <option key={user.username} value={user.username}>{displayUserLabel(user)}</option>
              ))}
            </select>
          </label>
          {selectedLedgerUser ? <button type="button" onClick={() => onLoadLedger(selectedLedgerUser)} disabled={ledgerLoading}>{ledgerLoading ? '刷新中...' : '刷新流水'}</button> : null}
        </div>
      </div>
      <div className="profile-table-wrap admin-ledger-table-wrap">
        <table className="admin-ledger-table">
          <thead>
            <tr>
              <th>变动</th>
              <th>类型</th>
              <th>原因</th>
              <th>管理员</th>
              <th>来源 ID</th>
              <th>时间</th>
            </tr>
          </thead>
          <tbody>
            {ledgerLoading ? (
              <tr><td colSpan={6}>正在读取流水...</td></tr>
            ) : !selectedLedgerUser ? (
              <tr><td colSpan={6}>请选择用户</td></tr>
            ) : ledger.length === 0 ? (
              <tr><td colSpan={6}>暂无流水</td></tr>
            ) : ledger.map((entry) => (
              <tr key={entry.id}>
                <td className={entry.delta >= 0 ? 'positive numeric-cell' : 'negative numeric-cell'}>{formatDelta(entry.delta)}</td>
                <td>{entry.type}</td>
                <td>{entry.reason || '-'}</td>
                <td>{entry.adminActor || '-'}</td>
                <td>{entry.sourceId || '-'}</td>
                <td>{formatDateTime(entry.createdAt)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  )
}
