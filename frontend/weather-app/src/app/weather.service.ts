import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { Weather } from './models/weather.model';

@Injectable({
  providedIn: 'root',
})
export class WeatherService {
  private apiUrl = 'http://localhost:8080/weather';

  constructor(private http: HttpClient) {}

  getWeather(city: string): Observable<Weather> {
    return this.http.get<Weather>(`${this.apiUrl}?city=${city}`);
  }

  getWeatherByCoords(lat: number, lon: number): Observable<Weather> {
    return this.http.get<Weather>(`${this.apiUrl}?lat=${lat}&lon=${lon}`);
  }
}
