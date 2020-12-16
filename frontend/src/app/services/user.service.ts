import { HttpClient } from "@angular/common/http";
import { Injectable } from "@angular/core";
import { map } from "rxjs/operators";
import { User } from "../models";

@Injectable({ providedIn: 'root' })
export class UserService {

    constructor(private http: HttpClient) {}

    public create(data: { name: string, email: string, password: string }) {
        return this.http.post<any>(`http://localhost:8000/auth/signup`, data)
        .pipe(map(resp => new User(resp)));
    }

    public get(id) {
        return {};
    }

}

