import { HttpClient } from "@angular/common/http";
import { Injectable } from "@angular/core";
import { map } from "rxjs/operators";

@Injectable({ providedIn: 'root' })
export class VideoService {

    constructor(private http: HttpClient) {}

    public create(data: { name: string, email: string, password: string }) {
        return this.http.post<any>(`http://localhost:8000/api/user`, data)
        .pipe(map(user => {

            return user;
        }));
    }


}

