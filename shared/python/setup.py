from setuptools import find_packages, setup

setup(
    name="aipx-shared",
    version="0.1.0",
    description="AIPX Shared Python Libraries",
    author="AIPX Team",
    packages=find_packages(),
    install_requires=[
        "protobuf>=4.24.0",
        "grpcio>=1.59.0",
        "grpcio-tools>=1.59.0",
        "kafka-python>=2.0.2",
        "confluent-kafka>=2.3.0",
        "redis>=5.0.0",
        "psycopg2-binary>=2.9.9",
        "SQLAlchemy>=2.0.0",
        "pydantic>=2.5.0",
        "pydantic-settings>=2.1.0",
        "python-dotenv>=1.0.0",
        "structlog>=23.2.0",
        "colorlog>=6.8.0",
        "aioredis>=2.0.1",
        "python-dateutil>=2.8.2",
        "pytz>=2023.3",
    ],
    python_requires=">=3.10",
)
