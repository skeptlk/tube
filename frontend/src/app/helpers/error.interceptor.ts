import { Injectable } from '@angular/core';
import { HttpRequest, HttpHandler, HttpEvent, HttpInterceptor } from '@angular/common/http';
import { Observable, throwError } from 'rxjs';
import { AuthService, NotificationService } from '../services';
import { catchError } from 'rxjs/operators';

@Injectable()
export class ErrorInterceptor implements HttpInterceptor {
    constructor(
        private authenticationService: AuthService, 
        private notifService: NotificationService
        ) { }

    intercept(request: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
        return next.handle(request).pipe(catchError(err => {
            if (err.status === 401) {
                // auto logout if 401 response returned from api
                this.authenticationService.logout();
                this.notifService.error("Not authorized!");
            }
            if (err.status === 404) {
                location.href = '/'; // TODO: replace with router.navigate
            }
            if (err.status !== 200) {
                const error = err.error?.message || err.statusText;
                console.log("Error intercepted!");
                
                return throwError(error);
            }
        }))
    }
}
