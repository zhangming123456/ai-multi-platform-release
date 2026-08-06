import json

from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    DATABASE_URL: str = "sqlite+aiosqlite:///./app.db"

    @property
    def resolved_db_url(self) -> str:
        if self.DATABASE_URL.startswith("sqlite"):
            import os

            base_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
            db_path = self.DATABASE_URL.split(":///")[-1]
            if db_path.startswith("./") or not db_path.startswith("/"):
                db_path = os.path.normpath(os.path.join(base_dir, db_path))
            return f"sqlite+aiosqlite:///{db_path}"
        return self.DATABASE_URL

    REDIS_URL: str = "redis://localhost:6379/0"
    SECRET_KEY: str = "your-secret-key-change-in-production"
    ACCESS_TOKEN_EXPIRE_MINUTES: int = 1440
    AI_API_KEY: str = ""
    AI_API_BASE_URL: str = "https://api.deepseek.com/v1"
    AI_MODEL: str = "deepseek-chat"
    CORS_ORIGINS: str = '["*"]'

    @property
    def cors_origins_list(self) -> list[str]:
        return json.loads(self.CORS_ORIGINS)

    model_config = {"env_file": ".env", "env_file_encoding": "utf-8"}


settings = Settings()
