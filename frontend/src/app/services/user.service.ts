import { HttpClient } from "@angular/common/http";
import { Injectable } from "@angular/core";
import { map } from "rxjs/operators";
import { User } from "../models";

@Injectable({ providedIn: 'root' })
export class UserService {

    BASE_URL: string = 'http://localhost:8000';

    constructor(private http: HttpClient) {}

    public create(data: { name: string, email: string, password: string }) {
        return this.http.post<any>(this.BASE_URL + `/auth/signup`, data)
            .pipe(map(resp => new User(resp)));
    }

    public get(id: number) {
        return this.http.get<any>(`${this.BASE_URL}/user/${id}`)
            .pipe(map(resp => new User(resp)));
    }

}

