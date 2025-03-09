# Use official Python image
FROM mcr.microsoft.com/playwright/python:v1.50.0

# Set the working directory
WORKDIR /app

# Copy the application files
COPY . /app

# Install dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Install Playwright browsers (if not already installed)
RUN playwright install chromium

# Set environment variables for Playwright
ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright
ENV PYTHONUNBUFFERED=1

# Command to run your Playwright script
CMD ["python", "main.py"]
