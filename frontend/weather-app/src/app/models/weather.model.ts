export interface Weather {
  city: string;
  country: string;
  localTime: string;
  humidity: number;
  temperature?: {
    celsius: number;
    fahrenheit: number;
  };
  wind?: {
    speed: number;
    deg: number;
  };
  condition: string;
  icon: string;
}
