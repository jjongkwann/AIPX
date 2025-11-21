"""User profile schema definitions."""

from datetime import datetime
from typing import Literal
from uuid import UUID

from pydantic import BaseModel, Field, field_validator


class UserProfile(BaseModel):
    """User investment profile."""

    user_id: UUID
    risk_tolerance: Literal["conservative", "moderate", "aggressive"] = Field(
        ..., description="User's risk tolerance level"
    )
    capital: int = Field(gt=0, description="Investment capital in USD")
    preferred_sectors: list[str] = Field(default_factory=list, description="Preferred investment sectors")
    investment_horizon: Literal["short", "medium", "long"] = Field(..., description="Investment time horizon")
    created_at: datetime = Field(default_factory=datetime.utcnow)
    updated_at: datetime = Field(default_factory=datetime.utcnow)

    @field_validator("preferred_sectors")
    @classmethod
    def validate_sectors(cls, v: list[str]) -> list[str]:
        """Validate preferred sectors list."""
        if not v:
            return v

        # Valid sectors list
        valid_sectors = {
            "Technology",
            "Healthcare",
            "Finance",
            "Energy",
            "Consumer",
            "Industrial",
            "Real Estate",
            "Materials",
            "Utilities",
            "Telecommunications",
        }

        invalid = set(v) - valid_sectors
        if invalid:
            raise ValueError(f"Invalid sectors: {invalid}. Valid sectors: {valid_sectors}")

        return v

    class Config:
        json_schema_extra = {
            "example": {
                "user_id": "550e8400-e29b-41d4-a716-446655440000",
                "risk_tolerance": "moderate",
                "capital": 100000,
                "preferred_sectors": ["Technology", "Healthcare"],
                "investment_horizon": "long",
            }
        }


class UserProfileCreate(BaseModel):
    """Schema for creating a new user profile."""

    risk_tolerance: Literal["conservative", "moderate", "aggressive"]
    capital: int = Field(gt=0)
    preferred_sectors: list[str] = Field(default_factory=list)
    investment_horizon: Literal["short", "medium", "long"]


class UserProfileUpdate(BaseModel):
    """Schema for updating user profile."""

    risk_tolerance: Literal["conservative", "moderate", "aggressive"] | None = None
    capital: int | None = Field(None, gt=0)
    preferred_sectors: list[str] | None = None
    investment_horizon: Literal["short", "medium", "long"] | None = None


class UserProfileResponse(BaseModel):
    """User profile response schema."""

    user_id: UUID
    risk_tolerance: str
    capital: int
    preferred_sectors: list[str]
    investment_horizon: str
    completeness: float  # 0-1 score indicating profile completeness
    missing_fields: list[str]

    @classmethod
    def from_profile(cls, profile: UserProfile) -> "UserProfileResponse":
        """Create response from profile."""
        missing = []
        fields_count = 0
        filled_count = 0

        for field in ["risk_tolerance", "capital", "investment_horizon"]:
            fields_count += 1
            if getattr(profile, field) is None:
                missing.append(field)
            else:
                filled_count += 1

        # Sectors are optional but count toward completeness if provided
        if profile.preferred_sectors:
            filled_count += 0.5
        fields_count += 0.5

        completeness = filled_count / fields_count if fields_count > 0 else 0.0

        return cls(
            user_id=profile.user_id,
            risk_tolerance=profile.risk_tolerance or "",
            capital=profile.capital or 0,
            preferred_sectors=profile.preferred_sectors or [],
            investment_horizon=profile.investment_horizon or "",
            completeness=completeness,
            missing_fields=missing,
        )
