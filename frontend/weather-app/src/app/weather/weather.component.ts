import { Component } from '@angular/core';
import { WeatherService } from '../weather.service';
import { Weather } from '../models/weather.model';

@Component({
  selector: 'app-weather',
  templateUrl: './weather.component.html',
  styleUrls: ['./weather.component.css'],
})
export class WeatherComponent {
  city: string = '';
  weatherData: Weather | null = null;

  constructor(private weatherService: WeatherService) {}

  errorMessage: string = '';
  loading: boolean = false;

  getWeather() {
    const queryCity = this.city?.trim();

    if (!queryCity) {
      this.errorMessage = 'Please enter a city name.';
      this.weatherData = null; // Hide old data
      return;
    }

    this.errorMessage = ''; // Clear previous error
    this.loading = true;
    this.weatherData = null; // Optional: hide data while loading

    this.weatherService.getWeather(queryCity).subscribe({
      next: (data) => {
        this.weatherData = data;
        this.loading = false;
      },
      error: (err) => {
        this.loading = false;
        this.errorMessage = 'City not found. Please try again.';
      },
    });
  }

  getCurrentLocation() {
    if (navigator.geolocation) {
      this.loading = true;
      this.errorMessage = ''; // Clear previous errors

      navigator.geolocation.getCurrentPosition(
        (position) => {
          const lat = position.coords.latitude;
          const lon = position.coords.longitude;

          // Call the new service method we just created
          this.weatherService.getWeatherByCoords(lat, lon).subscribe({
            next: (data) => {
              this.weatherData = data;
              this.loading = false;
              this.city = data.city; // Optional: update search box with the detected city name
            },
            error: (err) => {
              this.errorMessage =
                'Could not fetch weather for your coordinates.';
              this.loading = false;
            },
          });
        },
        (error) => {
          this.loading = false;
          // Handle common geolocation errors
          if (error.code === error.PERMISSION_DENIED) {
            this.errorMessage = 'Please allow location access in your browser.';
          } else {
            this.errorMessage = 'Location unavailable.';
          }
        }
      );
    } else {
      this.errorMessage = 'Geolocation is not supported by your browser.';
    }
  }

}