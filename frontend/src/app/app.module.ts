import { BrowserModule } from '@angular/platform-browser';
import { NgModule } from '@angular/core';
import { AppRoutingModule } from './app-routing.module';
import { ClarityModule } from '@clr/angular';
import { ReactiveFormsModule } from '@angular/forms';
import { FormsModule } from '@angular/forms';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';

import { HttpClientModule, HTTP_INTERCEPTORS } from '@angular/common/http';
import { JwtInterceptor, ErrorInterceptor } from './helpers';

import { AppComponent } from './app.component';
import { HomeComponent } from './home/home.component';
import { AdminComponent } from './admin/admin.component';
import { AccountComponent } from './account/account.component';
import { UploadComponent } from './upload/upload.component';
import { SearchComponent } from './search/search.component';
import { LoginComponent } from './login/login.component';
import { SignupComponent } from './signup/signup.component';
import { VideoComponent } from './video/video.component';
import { VideoOwnerControlsComponent } from './video-owner-controls/video-owner-controls.component';
import { CommentsComponent } from './comments/comments.component';
import { SingleCommentComponent } from './comments/single-comment/single-comment.component';


@NgModule({
    declarations: [
        AppComponent,
        HomeComponent,
        AdminComponent,
        AccountComponent,
        UploadComponent,
        SearchComponent,
        LoginComponent,
        SignupComponent,
        VideoComponent,
        VideoOwnerControlsComponent,
        CommentsComponent,
        SingleCommentComponent
    ],
    imports: [
        BrowserModule,
        AppRoutingModule,
        HttpClientModule,
        ClarityModule,
        BrowserAnimationsModule,
        ReactiveFormsModule,
        FormsModule
    ],
    providers: [
        { provide: HTTP_INTERCEPTORS, useClass: JwtInterceptor, multi: true },
        { provide: HTTP_INTERCEPTORS, useClass: ErrorInterceptor, multi: true },
    ],
    bootstrap: [AppComponent]
})
export class AppModule { };
