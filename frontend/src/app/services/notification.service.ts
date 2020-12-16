import { Injectable } from '@angular/core';

@Injectable({ providedIn: 'root' })
export class NotificationService {

    constructor() {}

    error(err: any) {
        // ...
    }

    message(msg: string) {
        alert(msg);
    }

}