export type PhotographyWeatherCredentialType = 'api_key' | 'token'

export interface PhotographyWeatherConfig {
  source: string
  name: string
  provider: string
  description: string
  api_key_masked: string
  api_host: string
  credential_type: PhotographyWeatherCredentialType
  configured: boolean
  enabled: boolean
  requires_key: boolean
  supports_host: boolean
}

export interface PhotographyWeatherConfigPayload {
  credential_type: PhotographyWeatherCredentialType
  api_key: string
  api_host: string
  clear_api_key: boolean
}
