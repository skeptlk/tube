import { HttpClient } from "@angular/common/http";
import { Injectable } from "@angular/core";
import { map } from "rxjs/operators";
import { Video, Category } from "../models";

@Injectable({ providedIn: 'root' })
export class VideoService {

    constructor(private http: HttpClient) { }

    public BASE_URL = "http://localhost:8000";

    public upload(data: any) {
        return this.http.post<any>(this.BASE_URL + `/api/video`, data).pipe();
    }

    public update(video: Video) {
        return this.http
            .put<any>(`${this.BASE_URL}/api/video/${video.id}`, video)
            .pipe(map(resp => new Video(resp)));
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

    // liked or disliked or none
    public likeInfo(id: number) {
        return this.http
            .get<any>(this.BASE_URL + `/api/like/` + id)
            .pipe()
    }

    public like(id: number) {
        return this.http
            .post<any>(this.BASE_URL + `/api/like/` + id, {})
            .pipe()
    }

    public removeLike(id: number) {
        return this.http
            .delete<any>(this.BASE_URL + `/api/like/` + id)
            .pipe()
    }

    public dislike(id: number) {
        return this.http
            .post<any>(this.BASE_URL + `/api/dislike/` + id, {})
            .pipe()
    }

    public removeDislike(id: number) {
        return this.http
            .delete<any>(this.BASE_URL + `/api/dislike/` + id)
            .pipe()
    }

    public getUserVideos(id: number) {
        return this.http.get<any>(`${this.BASE_URL}/user/${id}/video`)
            .pipe(
                map(resp => resp.map(vid => new Video(vid)))
            );
    }

    public getBestVideos() {
        return this.http.get<any>(`${this.BASE_URL}/v/best`)
            .pipe(
                map(resp => resp.map(vid => new Video(vid)))
            );
    }

    public getVideosInCategory(cat_id: number) {
        return this.http.get<any>(`${this.BASE_URL}/v/list?category=${cat_id}`)
            .pipe(
                map(resp => resp.map(vid => new Video(vid)))
            );
    }

    public getAllCategories() {
        return this.http.get<any>(`${this.BASE_URL}/api/category`)
            .pipe(
                map(resp => resp.map(cat => new Category(cat)))
            );
    }

}
