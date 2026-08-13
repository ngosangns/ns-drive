/** rclone config keys required to create (and auth) a remote of each type. */

export type RemoteConfigField = {
  key: string
  labelKey: string
  input: 'text' | 'password' | 'url'
  required: boolean
}

const USER_PASS: RemoteConfigField[] = [
  { key: 'user', labelKey: 'remotes.fields.user', input: 'text', required: true },
  { key: 'pass', labelKey: 'remotes.fields.pass', input: 'password', required: true },
]

const HOST_USER_PASS: RemoteConfigField[] = [
  { key: 'host', labelKey: 'remotes.fields.host', input: 'text', required: true },
  ...USER_PASS,
]

/** Fields to collect before create+test. Empty = no extra credentials (e.g. local). */
export function configFieldsFor(remoteType: string): RemoteConfigField[] {
  switch ((remoteType || '').trim()) {
    case 'mega':
      return USER_PASS
    case 'sftp':
    case 'ftp':
      return HOST_USER_PASS
    case 'webdav':
      return [
        { key: 'url', labelKey: 'remotes.fields.url', input: 'url', required: true },
        ...USER_PASS,
      ]
    case 's3':
      return [
        { key: 'provider', labelKey: 'remotes.fields.provider', input: 'text', required: false },
        { key: 'access_key_id', labelKey: 'remotes.fields.accessKey', input: 'text', required: true },
        { key: 'secret_access_key', labelKey: 'remotes.fields.secretKey', input: 'password', required: true },
        { key: 'region', labelKey: 'remotes.fields.region', input: 'text', required: false },
        { key: 'endpoint', labelKey: 'remotes.fields.endpoint', input: 'text', required: false },
      ]
    case 'b2':
      return [
        { key: 'account', labelKey: 'remotes.fields.account', input: 'text', required: true },
        { key: 'key', labelKey: 'remotes.fields.key', input: 'password', required: true },
      ]
    case 'crypt':
      return [
        { key: 'remote', labelKey: 'remotes.fields.wrappedRemote', input: 'text', required: true },
        { key: 'password', labelKey: 'remotes.fields.cryptPass', input: 'password', required: true },
      ]
    case 'alias':
      return [{ key: 'remote', labelKey: 'remotes.fields.wrappedRemote', input: 'text', required: true }]
    default:
      return []
  }
}

/** rclone `config create` argv pairs: `key=value`. */
export function toConfigKVs(values: Record<string, string>): string[] {
  const out: string[] = []
  for (const [key, raw] of Object.entries(values)) {
    const value = (raw ?? '').trim()
    if (!key || !value) continue
    out.push(`${key}=${value}`)
  }
  return out
}

export function missingRequiredFields(
  remoteType: string,
  values: Record<string, string>,
): string[] {
  return configFieldsFor(remoteType)
    .filter((f) => f.required && !(values[f.key] ?? '').trim())
    .map((f) => f.key)
}
