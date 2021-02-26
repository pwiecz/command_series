package cz.wie.p.command_series;

import androidx.appcompat.app.AppCompatActivity;

import android.os.Bundle;

import go.Seq;

import cz.wie.p.command_series.mobile.EbitenView;

public class MainActivity extends AppCompatActivity {

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);
        Seq.setContext(getApplicationContext());
    }

    private EbitenView getEbitenView() {
        return (EbitenView)this.findViewById(R.id.ebitenview);
    }

    @Override
    protected void onPause() {
        super.onPause();
        this.getEbitenView().suspendGame();
    }

    @Override
    protected void onResume() {
        super.onResume();
        this.getEbitenView().resumeGame();
    }
}
