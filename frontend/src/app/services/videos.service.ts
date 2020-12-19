import { HttpClient, HttpHeaders } from "@angular/common/http";
import { Injectable } from "@angular/core";
import { map } from "rxjs/operators";

@Injectable({ providedIn: 'root' })
export class VideoService {

    constructor(private http: HttpClient) {}

    public upload(data: any) {
        return this.http.post<any>(`http://localhost:8000/api/video`, data)
        .pipe(map(user => {

            return user;
        }));
    }


}

