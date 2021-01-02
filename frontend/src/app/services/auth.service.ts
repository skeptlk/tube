import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { BehaviorSubject, Observable } from 'rxjs';
import { map } from 'rxjs/operators';

import { User } from '../models';

@Injectable({ providedIn: 'root' })
export class AuthService {
    private currentUserSubject: BehaviorSubject<User>;
    public currentUser: Observable<User>;

    constructor(private http: HttpClient) {
        this.currentUserSubject = new BehaviorSubject<User>(JSON.parse(localStorage.getItem('currentUser')));
        this.currentUser = this.currentUserSubject.asObservable();
        
        console.log(this.currentUserSubject.value);
    }

    public get currentUserValue(): User {
        return this.currentUserSubject.value;
    }

    public get isAuthorized(): boolean {
        return !!(this.currentUserSubject.value);
    }

    public login(data : { name: string, password: string }) {
        return this.http.post<any>(`http://localhost:8000/auth/login`, data)
            .pipe(map(resp => {
                const user = resp.user;
                if (user) {
                    user.token = resp.token;
                    localStorage.setItem('currentUser', JSON.stringify(user));
                    this.currentUserSubject.next(user);
                    return user;
                }
            }));
    }

    public logout() {
        // remove user from local storage to log user out
        localStorage.removeItem('currentUser');
        this.currentUserSubject.next(null);
    }
}