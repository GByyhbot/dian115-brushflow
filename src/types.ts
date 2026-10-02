export interface Rules { free_only: boolean; exclude_hr: boolean; include: string; exclude: string; min_bytes: number; max_bytes: number; max_seeders: number; max_age_minutes: number; seed_minutes: number; seed_ratio: number; uploaded_bytes: number; download_minutes: number; delete_on_free_end: boolean }
export interface Site { id: string; name: string; feed_url: string; credential_ref: string }
export interface Downloader { id: string; name: string; base_url: string; credential_ref: string; save_path: string; category: string }
export interface Task { id: string; name: string; enabled: boolean; site_id: string; downloader_id: string; brush_minutes: number; check_minutes: number; max_active: number; notify: boolean; delete_files: boolean; rules: Rules }
export interface Config { schema_version: number; sites: Site[]; downloaders: Downloader[]; tasks: Task[] }
export interface RuntimeTask { last_brush?: string; last_check?: string; last_error?: string; records?: Record<string, { title: string; hash?: string; status: string; delete_reason?: string }> }
export interface State { config: Config; config_revision: string; mode: string; live_execution: boolean; version: string; heartbeat?: { at: string; enabled_tasks: number; status: string }; runtime?: { tasks: Record<string, RuntimeTask>; history: unknown[] } }
export interface Decision { candidate_id: string; accepted: boolean; reason: string }
export interface Result { status: 'succeeded' | 'failed' | 'accepted' | 'skipped'; message?: string; decisions?: Decision[] }
export interface Bridge {
 getState(view?: string): Promise<{ state?: State; etag?: string; not_modified?: boolean }>;
 invokeAction(action: string, input?: unknown): Promise<{ result?: Result }>;
 refresh(): Promise<State>;
}
export const emptyConfig = (): Config => ({ schema_version: 1, sites: [], downloaders: [], tasks: [] })
