import { HttpClient } from "@angular/common/http";
import { Injectable } from "@angular/core";
import { map } from "rxjs/operators";
import { Video } from "../models";

@Injectable({ providedIn: 'root' })
export class VideoService {

    constructor(private http: HttpClient) { }

    public BASE_URL = "http://localhost:8000";

    public upload(data: any) {
        return this.http.post<any>(this.BASE_URL + `/api/video`, data).pipe();
    }

    public delete(id: number) {
        return this.http.delete(this.BASE_URL + '/api/video/' + id).pipe();
    }

    public list() {
        return this.http
            .get<any>(this.BASE_URL + `/v/list`)
            .pipe(map(resp => resp.map(vid => new Video(vid))));
    }

    public filter(options: any) {
        return this.http
            .get<any>(this.BASE_URL + `/v/filter`, options)
            .pipe();
    }

    public getInfo(id: number) {
        return this.http.get<any>(this.BASE_URL + `/v/` + id)
            .pipe(map(resp => new Video(resp)))
    }

    public like() {

    }

    public removeLike() {

    }

    public dislike() {

    }

    public removeDislike() {
        
    }

}

