export interface Rules { free_only: boolean; exclude_hr: boolean; include: string; exclude: string; min_bytes: number; max_bytes: number }
export interface Task { id: string; name: string; enabled: boolean; site_id: string; downloader_id: string; brush_minutes: number; check_minutes: number; rules: Rules }
export interface Config { schema_version: number; tasks: Task[] }
export interface State { config: Config; config_revision: string; mode: string; live_execution: boolean; version: string; heartbeat?: { at: string; enabled_tasks: number; status: string } }
export interface Decision { candidate_id: string; accepted: boolean; reason: string }
export interface Result { status: 'succeeded' | 'failed' | 'accepted' | 'skipped'; message?: string; decisions?: Decision[] }
export interface Bridge {
 getState(view?: string): Promise<{ state?: State; etag?: string; not_modified?: boolean }>;
 invokeAction(action: string, input?: unknown): Promise<{ result?: Result }>;
 refresh(): Promise<State>;
}
export const emptyConfig = (): Config => ({ schema_version: 1, tasks: [] })
