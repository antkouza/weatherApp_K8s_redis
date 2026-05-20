import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: 'windDirection'
})
export class WindDirectionPipe implements PipeTransform {

  transform(deg:number | undefined): string {
    if (deg === undefined || deg === null) return '';
    
    // Define directions in clockwise order starting from North
    const direction = ['N', 'NE', 'E', 'SE', 'S', 'SW', 'W', 'NW'];

    // Split 360 degrees into 8 equal parts (45 degrees each) and find the index
    const index = Math.round(deg / 45) % 8;

    return direction[index];
  }
}
